package notify

import (
	"strings"
	"testing"
	"time"

	"nfetcher/internal/summary"
)

func TestFormatBarkBodyUsesHumanReadableStartedAt(t *testing.T) {
	result := summary.Result{
		Mode:      "run-once",
		Date:      "2026-04-04",
		StartedAt: time.Date(2026, 4, 4, 17, 30, 0, 0, time.FixedZone("CST", 8*60*60)),
	}

	body := formatBarkBody(result)

	if strings.Contains(body, "Date:") {
		t.Fatalf("expected Bark body to omit Date line, got %s", body)
	}

	if !strings.Contains(body, "At: 2026-04-04 17:30:00 CST") {
		t.Fatalf("expected Bark body to include At line, got %s", body)
	}
}
