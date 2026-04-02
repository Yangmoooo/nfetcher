package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nfetcher/internal/config"
	"nfetcher/internal/httpx"
	"nfetcher/internal/job"
	"nfetcher/internal/nhentai"
	"nfetcher/internal/scheduler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sharedHTTP := httpx.NewShared(cfg.HTTPTimeout, cfg.RequestRPS, cfg.RequestBurst, cfg.RetryMax)
	apiHTTP := httpx.NewClient(sharedHTTP, httpx.APIHeaders())
	imageHTTP := httpx.NewClient(sharedHTTP, httpx.ImageHeaders())
	nhClient := nhentai.NewClient(apiHTTP)

	logger := log.Default()
	runner := &job.Runner{
		Config:      cfg,
		Client:      nhClient,
		ImageClient: imageHTTP,
		Logger:      logger,
		NowFunc: func() time.Time {
			return time.Now().In(location)
		},
	}

	mode := cfg.RunMode
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "run-once":
		if err := runner.Run(ctx); err != nil {
			log.Fatal(err)
		}
	case "daemon":
		cronRunner, err := scheduler.Start(ctx, cfg.ScheduleCron, location, logger, runner)
		if err != nil {
			log.Fatal(err)
		}

		<-ctx.Done()
		stopCtx := cronRunner.Stop()
		<-stopCtx.Done()
	case "help", "--help", "-h":
		logger.Println("usage: nfetcher [run-once|daemon]")
	default:
		log.Fatalf("unknown mode: %s", mode)
	}
}
