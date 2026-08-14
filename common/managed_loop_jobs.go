package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/LukeEuler/dolly/log"
)

const (
	DefaultGracefulShutdownTimeout         = 30 * time.Second
	DefaultSingletonFailureShutdownTimeout = time.Second
)

var (
	ErrSingletonLoopStopped    = errors.New("singleton loop stopped unexpectedly")
	ErrLoopJobsShutdownTimeout = errors.New("loop jobs shutdown timed out")
)

// LoopContext receives cancellation and lets every loop request application
// shutdown before that loop has finished its own cleanup. RequestShutdown is
// idempotent for each loop. A nil reason requests a normal shutdown.
type LoopContext interface {
	context.Context
	RequestShutdown(reason error)
}

// ContextLoop requests application shutdown either through
// LoopContext.RequestShutdown or by returning. A non-nil return records an
// abnormal shutdown reason.
type ContextLoop func(ctx LoopContext) error

// SingletonLoopJobsOptions controls the two shutdown paths. Zero values use
// the exported defaults.
type SingletonLoopJobsOptions struct {
	GracefulShutdownTimeout         time.Duration
	SingletonFailureShutdownTimeout time.Duration
}

// LoopJobsShutdownTimeoutError reports that at least one loop did not return
// before its shutdown deadline. The process owner should log the error and
// terminate the process with a non-zero exit code.
type LoopJobsShutdownTimeoutError struct {
	Timeout time.Duration
	Cause   error
}

func (e *LoopJobsShutdownTimeoutError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%v after %s", ErrLoopJobsShutdownTimeout, e.Timeout)
	}
	return fmt.Sprintf("%v after %s: %v", ErrLoopJobsShutdownTimeout, e.Timeout, e.Cause)
}

func (e *LoopJobsShutdownTimeoutError) Unwrap() []error {
	if e.Cause == nil {
		return []error{ErrLoopJobsShutdownTimeout}
	}
	return []error{ErrLoopJobsShutdownTimeout, e.Cause}
}

type contextLoopRole uint8

const (
	ordinaryLoop contextLoopRole = iota
	singletonLoop
)

type contextLoopExit struct {
	kind contextLoopEventKind
	role contextLoopRole
	err  error
}

type contextLoopEventKind uint8

const (
	contextLoopShutdownRequest contextLoopEventKind = iota
	contextLoopExited
)

type managedLoopContext struct {
	context.Context
	requestOnce sync.Once
	role        contextLoopRole
	exits       chan<- contextLoopExit
}

func (c *managedLoopContext) RequestShutdown(reason error) {
	c.requestOnce.Do(func() {
		c.exits <- contextLoopExit{
			kind: contextLoopShutdownRequest,
			role: c.role,
			err:  reason,
		}
	})
}

// DoLoopJobsWithSingleton runs ordinary loops under the lifetime of a
// singleton loop. Ordinary shutdown stops and waits for all ordinary loops
// before stopping the singleton. If the singleton stops first, ordinary loops
// are cancelled immediately and get at most one second to return by default.
func DoLoopJobsWithSingleton(singleton ContextLoop, jobs ...ContextLoop) error {
	return DoLoopJobsWithSingletonOptions(SingletonLoopJobsOptions{}, singleton, jobs...)
}

// DoLoopJobsWithSingletonOptions is DoLoopJobsWithSingleton with configurable
// graceful and singleton-failure shutdown deadlines.
func DoLoopJobsWithSingletonOptions(options SingletonLoopJobsOptions, singleton ContextLoop, jobs ...ContextLoop) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	return doLoopJobsWithSingleton(options, singleton, signals, jobs...)
}

func doLoopJobsWithSingleton(
	options SingletonLoopJobsOptions,
	singleton ContextLoop,
	signals <-chan os.Signal,
	jobs ...ContextLoop,
) error {
	options, err := normalizeSingletonLoopJobsOptions(options)
	if err != nil {
		return err
	}
	if singleton == nil {
		return errors.New("singleton loop is nil")
	}
	for i, job := range jobs {
		if job == nil {
			return fmt.Errorf("ordinary loop %d is nil", i)
		}
	}

	ordinaryCtx, cancelOrdinary := context.WithCancel(context.Background())
	defer cancelOrdinary()
	singletonCtx, cancelSingleton := context.WithCancel(context.Background())
	defer cancelSingleton()

	// Every loop can produce at most one shutdown request and one exit event.
	exits := make(chan contextLoopExit, 2*(len(jobs)+1))
	startContextLoop(singletonCtx, singletonLoop, singleton, exits)
	for _, job := range jobs {
		startContextLoop(ordinaryCtx, ordinaryLoop, job, exits)
	}

	remainingOrdinary := len(jobs)
	singletonRunning := true
	singletonStopRequested := false
	shuttingDown := false
	emergencyShutdown := false
	var shutdownErr error
	var shutdownTimer *time.Timer
	var shutdownDeadline <-chan time.Time
	var activeTimeout time.Duration

	defer func() {
		if shutdownTimer != nil {
			shutdownTimer.Stop()
		}
	}()

	setDeadline := func(timeout time.Duration) {
		activeTimeout = timeout
		if shutdownTimer == nil {
			shutdownTimer = time.NewTimer(timeout)
		} else {
			if !shutdownTimer.Stop() {
				select {
				case <-shutdownTimer.C:
				default:
				}
			}
			shutdownTimer.Reset(timeout)
		}
		shutdownDeadline = shutdownTimer.C
	}

	requestSingletonStop := func() {
		if singletonRunning && !singletonStopRequested {
			singletonStopRequested = true
			cancelSingleton()
		}
	}

	beginGracefulShutdown := func(reason error) {
		shuttingDown = true
		shutdownErr = errors.Join(shutdownErr, reason)
		cancelOrdinary()
		setDeadline(options.GracefulShutdownTimeout)
		if remainingOrdinary == 0 {
			requestSingletonStop()
		}
	}

	beginEmergencyShutdown := func(reason error) {
		shuttingDown = true
		emergencyShutdown = true
		shutdownErr = errors.Join(shutdownErr, reason)
		cancelOrdinary()
		cancelSingleton()
		setDeadline(options.SingletonFailureShutdownTimeout)
	}

	for {
		if remainingOrdinary == 0 && singletonRunning && shuttingDown && !emergencyShutdown {
			requestSingletonStop()
		}
		if remainingOrdinary == 0 && !singletonRunning {
			return shutdownErr
		}

		select {
		case sig := <-signals:
			if !shuttingDown {
				log.Entry.Infof("received signal [%v], preparing to quit", sig)
				beginGracefulShutdown(nil)
			}

		case exit := <-exits:
			if exit.kind == contextLoopShutdownRequest {
				if exit.role == singletonLoop {
					reason := exit.err
					if reason == nil {
						reason = ErrSingletonLoopStopped
					}
					if !emergencyShutdown {
						beginEmergencyShutdown(reason)
					} else {
						shutdownErr = appendLoopError(shutdownErr, reason)
					}
				} else if !shuttingDown {
					beginGracefulShutdown(exit.err)
				} else if !isExpectedCancellation(exit.err) {
					shutdownErr = appendLoopError(shutdownErr, exit.err)
				}
				continue
			}

			switch exit.role {
			case ordinaryLoop:
				remainingOrdinary--
				if !shuttingDown {
					beginGracefulShutdown(exit.err)
				} else if !isExpectedCancellation(exit.err) {
					shutdownErr = appendLoopError(shutdownErr, exit.err)
				}

			case singletonLoop:
				singletonRunning = false
				if singletonStopRequested {
					if !isExpectedCancellation(exit.err) {
						shutdownErr = appendLoopError(shutdownErr, exit.err)
					}
					continue
				}
				if emergencyShutdown {
					if !isExpectedCancellation(exit.err) {
						shutdownErr = appendLoopError(shutdownErr, exit.err)
					}
					continue
				}

				reason := exit.err
				if reason == nil {
					reason = ErrSingletonLoopStopped
				}
				beginEmergencyShutdown(reason)
			}

		case <-shutdownDeadline:
			cancelOrdinary()
			cancelSingleton()
			return &LoopJobsShutdownTimeoutError{
				Timeout: activeTimeout,
				Cause:   shutdownErr,
			}
		}
	}
}

func normalizeSingletonLoopJobsOptions(options SingletonLoopJobsOptions) (SingletonLoopJobsOptions, error) {
	if options.GracefulShutdownTimeout < 0 {
		return options, errors.New("graceful shutdown timeout must not be negative")
	}
	if options.SingletonFailureShutdownTimeout < 0 {
		return options, errors.New("singleton failure shutdown timeout must not be negative")
	}
	if options.GracefulShutdownTimeout == 0 {
		options.GracefulShutdownTimeout = DefaultGracefulShutdownTimeout
	}
	if options.SingletonFailureShutdownTimeout == 0 {
		options.SingletonFailureShutdownTimeout = DefaultSingletonFailureShutdownTimeout
	}
	return options, nil
}

func startContextLoop(ctx context.Context, role contextLoopRole, loop ContextLoop, exits chan<- contextLoopExit) {
	loopCtx := &managedLoopContext{
		Context: ctx,
		role:    role,
		exits:   exits,
	}
	go func() {
		var err error
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("loop panic: %v\nstacktrace from panic: %s", recovered, string(debug.Stack()))
			}
			exits <- contextLoopExit{kind: contextLoopExited, role: role, err: err}
		}()
		err = loop(loopCtx)
	}()
}

func appendLoopError(current, next error) error {
	if next == nil || isExpectedCancellation(next) {
		return current
	}
	return errors.Join(current, next)
}

func isExpectedCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
