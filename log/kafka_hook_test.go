package log

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestKafkaFormatterUsesMillisecondTimestamp(t *testing.T) {
	entryTime := time.Date(2026, time.October, 20, 16, 31, 37, 504438001, time.FixedZone("CST", 8*60*60))
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Data:    logrus.Fields{},
		Time:    entryTime,
		Level:   logrus.InfoLevel,
		Message: "test",
	}

	formatted, err := newKafkaJSONFormatter().Format(entry)
	if err != nil {
		t.Fatalf("format Kafka log entry: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(formatted, &fields); err != nil {
		t.Fatalf("decode formatted Kafka log entry: %v", err)
	}

	const want = "2026-10-20T16:31:37.504+08:00"
	if got := fields[logrus.FieldKeyTime]; got != want {
		t.Fatalf("unexpected Kafka timestamp: got %q, want %q", got, want)
	}
}
