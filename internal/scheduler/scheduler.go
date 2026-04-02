package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type Runner interface {
	Run(context.Context) error
}

func Start(ctx context.Context, spec string, location *time.Location, logger *log.Logger, runner Runner) (*cron.Cron, error) {
	cronLogger := cron.PrintfLogger(logger)
	c := cron.New(
		cron.WithLocation(location),
		cron.WithChain(
			cron.SkipIfStillRunning(cronLogger),
			cron.Recover(cronLogger),
		),
	)

	_, err := c.AddFunc(spec, func() {
		if err := runner.Run(ctx); err != nil {
			logger.Printf("scheduled job failed: %v", err)
		}
	})
	if err != nil {
		return nil, err
	}

	c.Start()
	return c, nil
}
