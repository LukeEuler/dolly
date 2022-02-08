package log

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var Entry *logrus.Entry

func init() {
	Entry = logrus.NewEntry(logrus.New())
	Entry.Logger.SetReportCaller(true)
	Entry.Logger.SetLevel(logrus.DebugLevel)
	Entry.Logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		CallerPrettyfier: callerPrettyfier,
	})
	// Entry.Data["devlang"] = "golang"
}

func AddField(key, value string) {
	if len(key) == 0 {
		return
	}
	if len(value) == 0 {
		return
	}
	Entry = Entry.WithField(key, value)
}

func callerPrettyfier(f *runtime.Frame) (string, string) {
	fileName := fmt.Sprintf("%s:%d", f.File, f.Line)
	funcName := f.Function
	list := strings.Split(funcName, "/")
	if len(list) > 0 {
		funcName = list[len(list)-1]
	}
	return funcName, fileName
}

// for stdout
func callerFormatter(f *runtime.Frame) string {
	funcName, fileName := callerPrettyfier(f)
	return " @" + funcName + " " + fileName
}

// DisableDefaultConsole 取消默认的控制台输出
func DisableDefaultConsole() {
	Entry.Logger.SetOutput(io.Discard)
}

func getHookLevel(level int) []logrus.Level {
	if level < 0 || level > 5 {
		level = 5
	}
	return logrus.AllLevels[:level+1]
}
