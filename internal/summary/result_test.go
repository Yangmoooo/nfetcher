package summary

import (
	"strings"
	"testing"
	"time"
)

func TestResultLogLineIncludesHumanReadableStartedAt(t *testing.T) {
	result := Result{
		Mode:      "run-once",
		Date:      "2026-04-04",
		Query:     "language:chinese",
		Sort:      "popular-today",
		Page:      1,
		StartedAt: time.Date(2026, 4, 4, 17, 30, 0, 0, time.FixedZone("CST", 8*60*60)),
	}

	logLine := result.LogLine()

	if !strings.Contains(logLine, "date=2026-04-04") {
		t.Fatalf("expected business date in log line, got %s", logLine)
	}

	if !strings.Contains(logLine, `at="2026-04-04 17:30:00 CST"`) {
		t.Fatalf("expected human readable started-at in log line, got %s", logLine)
	}
}
