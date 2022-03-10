package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// AddFileOut only support unix os
func AddFileOut(logFilePath string, level, days int) error {
	absPath, err := filepath.Abs(logFilePath)
	if err != nil {
		return errors.WithStack(err)
	}

	logDirPath := filepath.Dir(absPath)
	if _, err = os.Stat(logDirPath); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(logDirPath, 0755); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err != nil {
		return errors.WithStack(err)
	}
	if err = unix.Access(logDirPath, unix.W_OK); err != nil {
		return errors.Wrapf(err, "%s is not writable", logDirPath)
	}

	var logf *rotatelogs.RotateLogs
	logf, err = rotatelogs.New(
		absPath+".%Y-%m-%dT%H:%M",
		rotatelogs.WithLinkName(absPath),
		rotatelogs.WithMaxAge(time.Duration(days)*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err = errors.WithStack(err); err != nil {
		return err
	}
	hook := &fileHook{
		formatter: &logrus.TextFormatter{
			DisableTimestamp: false,
			CallerPrettyfier: callerPrettyfier,
		},
		levels: getHookLevel(level),
		rotate: logf,
	}

	Entry.Logger.AddHook(hook)
	return nil
}

type fileHook struct {
	formatter logrus.Formatter
	levels    []logrus.Level
	rotate    *rotatelogs.RotateLogs
}

func (hook *fileHook) Fire(entry *logrus.Entry) error {
	formatBytes, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = hook.rotate.Write(formatBytes)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unable to write file on filehook(entry.String)%v", err)
		return err
	}
	return nil
}

func (hook *fileHook) Levels() []logrus.Level {
	return hook.levels
}
