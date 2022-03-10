package log

import (
	"fmt"
	"runtime/debug"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Error print stack trace if possible
func Error(err error) {
	ErrorWithFields(err, nil)
}

// ErrorWithFields print stack trace if possible
func ErrorWithFields(err error, inFields logrus.Fields) {
	fields := getFieldsCopy(inFields)
	if err == nil {
		Entry.WithFields(fields).Errorf("nil error!!!\n%s", debug.Stack())
		// // 这个方法是从 errors.WithStack() 抄来的，但是没有 debug.Stack() 简单，而且看不到子协程的来源
		// const depth = 32
		// var pcs [depth]uintptr
		// n := runtime.Callers(1, pcs[:])
		// var info string
		// lf := make([]errors.Frame, n)
		// for i := 0; i < n; i++ {
		// 	lf[i] = errors.Frame(pcs[i])
		// }
		// for _, f := range lf {
		// 	info += fmt.Sprintf("\n%+v", f)
		// }
		// Entry.WithFields(fields).Errorf("nil error!!!\n%s", info)
		return
	}

	type causer interface {
		Cause() error
	}

	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	info := err.Error()
	var stack string
	var st stackTracer
	for {
		if _, ok := err.(stackTracer); ok {
			st = err.(stackTracer)
		}
		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {
			break
		}
	}

	if st != nil {
		for _, f := range st.StackTrace() {
			stack += fmt.Sprintf("\n%+v", f)
		}
		if fields == nil {
			fields = make(logrus.Fields)
		}
		fields["stack"] = stack

		Entry.WithFields(fields).Error(info)
		return
	}
	Entry.WithFields(fields).Errorf("%#v", err)
}

// Fatal = Error + exit(1)
func Fatal(err error) {
	Error(err)
	Entry.Logger.Exit(1)
}

func getFieldsCopy(in logrus.Fields) (out logrus.Fields) {
	if len(in) != 0 {
		out = make(logrus.Fields, len(in))
	} else {
		out = make(logrus.Fields)
	}
	for k, v := range in {
		out[k] = v
	}
	return
}
