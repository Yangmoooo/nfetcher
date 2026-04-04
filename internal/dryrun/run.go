package dryrun

import (
	"context"
	"errors"
	"log"
	"time"

	"nfetcher/internal/config"
	"nfetcher/internal/job"
	"nfetcher/internal/notify"
	"nfetcher/internal/storage"
	"nfetcher/internal/summary"
)

type Executor struct {
	Config   config.Config
	Runner   *job.Runner
	Logger   *log.Logger
	Mode     string
	Notifier notify.Sender
}

func (e *Executor) Run(ctx context.Context) (runErr error) {
	startedAt := time.Now()
	now := e.now()
	day := now.Format("2006-01-02")
	result := summary.Result{
		Mode:      e.mode(),
		Date:      day,
		StartedAt: now,
		Query:     e.Config.SearchQuery,
		Sort:      e.Config.SearchSort,
		Page:      e.Config.SearchPage,
	}
	defer func() {
		result.Duration = time.Since(startedAt)
		e.finalizeRun(ctx, result)
	}()

	e.logger().Printf("dry-run start date=%s at=%q query=%q sort=%q page=%d", day, result.StartedAtText(), e.Config.SearchQuery, e.Config.SearchSort, e.Config.SearchPage)

	report := RunPreflight(e.Config)
	result.PreflightWarnings = report.WarningCount()
	result.PreflightFailures = report.FailureCount()
	for _, check := range report.Checks {
		e.logger().Printf("preflight status=%s check=%s detail=%q", check.Status, check.Name, check.Detail)
	}

	plan, err := e.Runner.BuildPlan(ctx, job.PlanOptions{Log: false})
	if err != nil {
		result.ErrorCount = report.FailureCount() + 1
		runErr = errors.Join(report.Failure(), err)
		return
	}
	result.SearchResults = plan.SearchResultsCount
	result.Duplicates = len(plan.Duplicates)
	result.Queued = len(plan.Queued)
	result.DetailErrors = len(plan.Errors)
	result.FailedGalleryIDs = append(result.FailedGalleryIDs, plan.DetailFailedIDs...)

	e.logger().Printf(
		"plan summary date=%s at=%q query=%q sort=%q page=%d search_results=%d duplicates=%d queued=%d detail_errors=%d",
		day,
		result.StartedAtText(),
		e.Config.SearchQuery,
		e.Config.SearchSort,
		e.Config.SearchPage,
		plan.SearchResultsCount,
		len(plan.Duplicates),
		len(plan.Queued),
		len(plan.Errors),
	)

	for _, duplicate := range plan.Duplicates {
		e.logger().Printf(
			"plan skip rank=%d gallery_id=%d pages=%d title=%q reason=duplicate existing_path=%q",
			duplicate.Rank,
			duplicate.Gallery.ID,
			len(duplicate.Gallery.Pages),
			storage.ChooseTitle(duplicate.Gallery),
			duplicate.ExistingPath,
		)
	}

	for queueIndex, queued := range plan.Queued {
		e.logger().Printf(
			"plan queue order=%d rank=%d gallery_id=%d pages=%d title=%q reason=download",
			queueIndex+1,
			queued.Rank,
			queued.Gallery.ID,
			len(queued.Gallery.Pages),
			storage.ChooseTitle(queued.Gallery),
		)
	}

	for _, planErr := range plan.Errors {
		e.logger().Printf("plan detail_error error=%v", planErr)
	}

	e.logger().Printf("dry-run finish date=%s at=%q queued=%d duplicates=%d", day, result.StartedAtText(), len(plan.Queued), len(plan.Duplicates))

	result.ErrorCount = report.FailureCount() + len(plan.Errors)
	runErr = errors.Join(report.Failure(), errors.Join(plan.Errors...))
	return
}

func (e *Executor) logger() *log.Logger {
	if e.Logger != nil {
		return e.Logger
	}
	return log.Default()
}

func (e *Executor) now() time.Time {
	if e.Runner != nil && e.Runner.NowFunc != nil {
		return e.Runner.NowFunc()
	}
	return time.Now()
}

func (e *Executor) mode() string {
	if e.Mode != "" {
		return e.Mode
	}
	return "dry-run"
}

func (e *Executor) finalizeRun(ctx context.Context, result summary.Result) {
	e.logger().Print(result.LogLine())
	if e.Notifier == nil {
		return
	}
	if err := e.Notifier.Send(ctx, result); err != nil {
		e.logger().Printf("notify bark failed error=%v", err)
	}
}
