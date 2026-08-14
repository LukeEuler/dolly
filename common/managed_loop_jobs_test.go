package common

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoopJobsWithSingletonStopsSingletonLast(t *testing.T) {
	t.Parallel()

	var mutex sync.Mutex
	var stopped []string
	recordStop := func(name string) {
		mutex.Lock()
		defer mutex.Unlock()
		stopped = append(stopped, name)
	}

	singleton := func(ctx LoopContext) error {
		<-ctx.Done()
		recordStop("singleton")
		return nil
	}
	worker := func(ctx LoopContext) error {
		<-ctx.Done()
		recordStop("worker")
		return nil
	}
	requestShutdown := func(ctx LoopContext) error {
		recordStop("requester")
		ctx.RequestShutdown(nil)
		<-ctx.Done()
		return nil
	}

	err := doLoopJobsWithSingleton(
		SingletonLoopJobsOptions{},
		singleton,
		nil,
		worker,
		requestShutdown,
	)
	require.NoError(t, err)
	assert.Equal(t, "singleton", stopped[len(stopped)-1])
	assert.ElementsMatch(t, []string{"requester", "worker", "singleton"}, stopped)
}

func TestLoopJobsWithSingletonHandlesSignalGracefully(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 1)
	workerStopped := make(chan struct{})
	singletonSawWorkerStop := make(chan bool, 1)

	singleton := func(ctx LoopContext) error {
		<-ctx.Done()
		select {
		case <-workerStopped:
			singletonSawWorkerStop <- true
		default:
			singletonSawWorkerStop <- false
		}
		return nil
	}
	worker := func(ctx LoopContext) error {
		<-ctx.Done()
		close(workerStopped)
		return ctx.Err()
	}

	result := make(chan error, 1)
	go func() {
		result <- doLoopJobsWithSingleton(SingletonLoopJobsOptions{}, singleton, signals, worker)
	}()

	signals <- os.Interrupt
	require.NoError(t, <-result)
	assert.True(t, <-singletonSawWorkerStop)
}

func TestLoopJobsWithSingletonCancelsWorkersWhenSingletonFails(t *testing.T) {
	t.Parallel()

	lockErr := errors.New("singleton heartbeat failed")
	workerCancelled := make(chan struct{})
	singleton := func(ctx LoopContext) error {
		ctx.RequestShutdown(lockErr)
		// The request must cancel workers before the singleton loop returns.
		<-workerCancelled
		return lockErr
	}
	worker := func(ctx LoopContext) error {
		<-ctx.Done()
		close(workerCancelled)
		return nil
	}

	err := doLoopJobsWithSingleton(SingletonLoopJobsOptions{}, singleton, nil, worker)
	require.ErrorIs(t, err, lockErr)
	assertClosed(t, workerCancelled)
}

func TestLoopJobsWithSingletonTimesOutStuckWorkerAfterSingletonFailure(t *testing.T) {
	t.Parallel()

	lockErr := errors.New("singleton heartbeat failed")
	emergencyTimeout := 40 * time.Millisecond
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	singletonMayFail := make(chan struct{})

	singleton := func(ctx LoopContext) error {
		<-singletonMayFail
		ctx.RequestShutdown(lockErr)
		return lockErr
	}
	worker := func(LoopContext) error {
		close(workerStarted)
		<-releaseWorker
		return nil
	}

	result := make(chan error, 1)
	startedAt := time.Now()
	go func() {
		result <- doLoopJobsWithSingleton(
			SingletonLoopJobsOptions{SingletonFailureShutdownTimeout: emergencyTimeout},
			singleton,
			nil,
			worker,
		)
	}()
	assertClosed(t, workerStarted)
	close(singletonMayFail)

	err := <-result
	close(releaseWorker)
	require.ErrorIs(t, err, ErrLoopJobsShutdownTimeout)
	require.ErrorIs(t, err, lockErr)
	assert.GreaterOrEqual(t, time.Since(startedAt), emergencyTimeout)
}

func TestLoopJobsWithSingletonEscalatesGracefulShutdown(t *testing.T) {
	t.Parallel()

	lockErr := errors.New("singleton heartbeat failed during shutdown")
	emergencyTimeout := 40 * time.Millisecond
	workerCancelled := make(chan struct{})
	releaseWorker := make(chan struct{})
	failSingleton := make(chan struct{})

	singleton := func(ctx LoopContext) error {
		<-failSingleton
		ctx.RequestShutdown(lockErr)
		return lockErr
	}
	stuckWorker := func(ctx LoopContext) error {
		<-ctx.Done()
		close(workerCancelled)
		<-releaseWorker
		return nil
	}
	requestShutdown := func(ctx LoopContext) error {
		ctx.RequestShutdown(nil)
		<-ctx.Done()
		return ctx.Err()
	}

	result := make(chan error, 1)
	go func() {
		result <- doLoopJobsWithSingleton(
			SingletonLoopJobsOptions{
				GracefulShutdownTimeout:         time.Second,
				SingletonFailureShutdownTimeout: emergencyTimeout,
			},
			singleton,
			nil,
			stuckWorker,
			requestShutdown,
		)
	}()

	assertClosed(t, workerCancelled)
	escalatedAt := time.Now()
	close(failSingleton)
	err := <-result
	close(releaseWorker)

	require.ErrorIs(t, err, ErrLoopJobsShutdownTimeout)
	require.ErrorIs(t, err, lockErr)
	assert.GreaterOrEqual(t, time.Since(escalatedAt), emergencyTimeout)
	assert.Less(t, time.Since(escalatedAt), 500*time.Millisecond)
}

func TestLoopJobsWithSingletonConvertsLoopPanicToError(t *testing.T) {
	t.Parallel()

	singleton := func(ctx LoopContext) error {
		<-ctx.Done()
		return nil
	}
	panickingWorker := func(LoopContext) error {
		panic("worker failed")
	}

	err := doLoopJobsWithSingleton(SingletonLoopJobsOptions{}, singleton, nil, panickingWorker)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker failed")
}

func TestLoopJobsWithSingletonRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	noop := func(LoopContext) error { return nil }

	err := doLoopJobsWithSingleton(
		SingletonLoopJobsOptions{SingletonFailureShutdownTimeout: -time.Second},
		noop,
		nil,
	)
	require.EqualError(t, err, "singleton failure shutdown timeout must not be negative")

	err = doLoopJobsWithSingleton(SingletonLoopJobsOptions{}, nil, nil)
	require.EqualError(t, err, "singleton loop is nil")

	err = doLoopJobsWithSingleton(SingletonLoopJobsOptions{}, noop, nil, nil)
	require.EqualError(t, err, "ordinary loop 0 is nil")
}

func assertClosed(t *testing.T, channel <-chan struct{}) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(time.Second):
		t.Fatal("channel was not closed")
	}
}
