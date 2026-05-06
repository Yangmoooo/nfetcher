package job

import (
	"context"
	"testing"
	"time"

	"nfetcher/internal/config"
)

func TestWaitForDownloadIssueSlotHonorsConfiguredInterval(t *testing.T) {
	runner := &Runner{
		Config: config.Config{
			DownloadIssueInterval: 20 * time.Millisecond,
		},
	}

	if err := runner.waitForDownloadIssueSlot(context.Background()); err != nil {
		t.Fatalf("first wait: %v", err)
	}

	startedAt := time.Now()
	if err := runner.waitForDownloadIssueSlot(context.Background()); err != nil {
		t.Fatalf("second wait: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed < runner.Config.DownloadIssueInterval {
		t.Fatalf("expected second wait to honor interval, elapsed %s", elapsed)
	}
}

func TestWaitForDownloadIssueSlotCanBeDisabled(t *testing.T) {
	runner := &Runner{
		Config: config.Config{
			DownloadIssueInterval: 0,
		},
	}

	startedAt := time.Now()
	if err := runner.waitForDownloadIssueSlot(context.Background()); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 10*time.Millisecond {
		t.Fatalf("expected disabled interval to return immediately, elapsed %s", elapsed)
	}
}
