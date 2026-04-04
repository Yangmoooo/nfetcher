package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nfetcher/internal/config"
	"nfetcher/internal/dryrun"
	"nfetcher/internal/httpx"
	"nfetcher/internal/job"
	"nfetcher/internal/nhentai"
	"nfetcher/internal/notify"
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
	notifyHTTP := httpx.NewClient(sharedHTTP, httpx.APIHeaders())
	nhClient := nhentai.NewClient(apiHTTP)
	barkNotifier := notify.NewBark(notifyHTTP, cfg)

	logger := log.Default()
	runner := &job.Runner{
		Config:      cfg,
		Client:      nhClient,
		ImageClient: imageHTTP,
		Logger:      logger,
		Notifier:    barkNotifier,
		NowFunc: func() time.Time {
			return time.Now().In(location)
		},
	}
	dryRunner := &dryrun.Executor{
		Config:   cfg,
		Runner:   runner,
		Logger:   logger,
		Notifier: barkNotifier,
	}

	mode := cfg.RunMode
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "run-once":
		runner.Mode = "run-once"
		if err := runner.Run(ctx); err != nil {
			log.Fatal(err)
		}
	case "dry-run":
		dryRunner.Mode = "dry-run"
		if err := dryRunner.Run(ctx); err != nil {
			log.Fatal(err)
		}
	case "daemon":
		runner.Mode = "daemon"
		cronRunner, err := scheduler.Start(ctx, cfg.ScheduleCron, location, logger, runner)
		if err != nil {
			log.Fatal(err)
		}

		<-ctx.Done()
		stopCtx := cronRunner.Stop()
		<-stopCtx.Done()
	case "help", "--help", "-h":
		logger.Println("usage: nfetcher [run-once|dry-run|daemon]")
	default:
		log.Fatalf("unknown mode: %s", mode)
	}
}
