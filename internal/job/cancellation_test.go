package job

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"nfetcher/internal/config"
	"nfetcher/internal/nhentai"
)

type canceledDoer struct{}

func (canceledDoer) Do(ctx context.Context, _ *http.Request) (*http.Response, error) {
	return nil, ctx.Err()
}

func TestBuildPlanPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := &Runner{
		Client: nhentai.NewClient(canceledDoer{}),
		Config: config.Config{
			SearchQuery: "language:chinese",
			SearchSort:  "popular-today",
			SearchPage:  1,
		},
	}

	_, err := runner.BuildPlan(ctx, PlanOptions{ExistingGalleryPaths: map[int64]string{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
