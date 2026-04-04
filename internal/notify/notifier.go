package notify

import (
	"context"

	"nfetcher/internal/summary"
)

type Sender interface {
	Send(context.Context, summary.Result) error
}
