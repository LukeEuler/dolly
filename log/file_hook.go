package log

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func AddFileOut(logFilePath string, level, maxSize, maxBackups, maxAge int) error {
	writer := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    maxSize, // MB
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		LocalTime:  true,
		Compress:   true,
	}

	formatter := &logrus.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  time.StampMilli,
		CallerPrettyfier: callerPrettyfier,
	}

	levels := getHookLevel(level)

	hook := &lumberjackHook{
		writer:    writer,
		formatter: formatter,
		levels:    levels,
	}

	Entry.Logger.AddHook(hook)
	return nil
}

type lumberjackHook struct {
	writer    *lumberjack.Logger
	formatter logrus.Formatter
	levels    []logrus.Level
	mu        sync.Mutex
}

func (h *lumberjackHook) Levels() []logrus.Level {
	return h.levels
}

func (h *lumberjackHook) Fire(entry *logrus.Entry) error {
	// formatter 不是并发安全的
	h.mu.Lock()
	defer h.mu.Unlock()

	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	_, err = h.writer.Write(line)
	return err
}
