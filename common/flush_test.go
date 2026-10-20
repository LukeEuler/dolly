package common

import (
	"strings"
	"testing"
	"time"
)

func TestFormatHeartBeatStartTimeUsesSecondPrecisionAndTimezone(t *testing.T) {
	start := time.Date(2026, time.October, 20, 16, 50, 18, 198397000, time.FixedZone("CST", 8*60*60))

	const want = "2026-10-20 16:50:18 +0800 CST"
	got := formatHeartBeatStartTime(start)
	if got != want {
		t.Fatalf("unexpected heartbeat start time: got %q, want %q", got, want)
	}
	if strings.Contains(got, ".") || strings.Contains(got, "m=+") {
		t.Fatalf("heartbeat start time contains subsecond or monotonic data: %q", got)
	}
}
