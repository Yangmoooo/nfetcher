package job

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nfetcher/internal/archive"
	"nfetcher/internal/config"
	"nfetcher/internal/nhentai"
	"nfetcher/internal/notify"
	"nfetcher/internal/storage"
	"nfetcher/internal/summary"
)

type Runner struct {
	Config         config.Config
	Client         *nhentai.Client
	DownloadClient FileDownloader
	Logger         *log.Logger
	NowFunc        func() time.Time
	Mode           string
	Notifier       notify.Sender

	downloadIssueMu     sync.Mutex
	lastDownloadIssueAt time.Time
}

func (r *Runner) Run(ctx context.Context) (runErr error) {
	startedAt := time.Now()
	now := r.now()
	day := now.Format("2006-01-02")
	storyArc := formatStoryArc(day)
	runSummary := summary.Result{
		Mode:      r.mode(),
		Date:      day,
		StartedAt: now,
		Query:     r.Config.SearchQuery,
		Sort:      r.Config.SearchSort,
		Page:      r.Config.SearchPage,
	}
	defer func() {
		runSummary.Duration = time.Since(startedAt)
		r.finalizeRun(ctx, runSummary)
	}()

	r.logger().Printf("job start date=%s at=%q query=%q sort=%q page=%d", day, runSummary.StartedAtText(), r.Config.SearchQuery, r.Config.SearchSort, r.Config.SearchPage)

	index, err := storage.ScanLibraryIndex(r.Config.LibraryPath())
	if err != nil {
		runSummary.ErrorCount = 1
		runErr = fmt.Errorf("scan library index: %w", err)
		return
	}

	plan, err := r.BuildPlan(ctx, PlanOptions{
		Log:                  true,
		ExistingGalleryPaths: index.ExistingGalleryPaths(),
	})
	if err != nil {
		runSummary.ErrorCount = 1
		runErr = err
		return
	}
	runSummary.SearchResults = plan.SearchResultsCount
	runSummary.Duplicates = len(plan.Duplicates)
	runSummary.Queued = len(plan.Queued)

	runErrors := make([]error, 0)
	for processResult := range ProcessGalleries(ctx, plan.Queued, r.Config.GalleryConcurrency, func(workerCtx context.Context, gallery QueuedGallery) error {
		return r.processGallery(workerCtx, storyArc, gallery)
	}) {
		if processResult.Err != nil {
			runErrors = append(runErrors, fmt.Errorf("process gallery %d: %w", processResult.QueuedGallery.Gallery.ID, processResult.Err))
			r.logger().Printf("gallery archive failed gallery_id=%d error=%v", processResult.QueuedGallery.Gallery.ID, processResult.Err)
			runSummary.ArchivedFailed++
			runSummary.FailedGalleryIDs = append(runSummary.FailedGalleryIDs, processResult.QueuedGallery.Gallery.ID)
			continue
		}

		runSummary.ArchivedOK++
		finalPath := storage.FinalGalleryPath(r.Config.LibraryPath(), storage.GalleryFileName(processResult.QueuedGallery.Gallery))
		index.Archives = append(index.Archives, storage.LibraryArchive{
			Path:        finalPath,
			GalleryID:   processResult.QueuedGallery.Gallery.ID,
			FetchedDate: day,
		})
	}
	if err := ctx.Err(); err != nil {
		runErrors = append(runErrors, err)
		runSummary.ErrorCount = len(runErrors)
		runErr = errors.Join(runErrors...)
		return
	}

	removedFiles, err := CleanupExpired(&index, now, r.Config.RetentionDays)
	if err != nil {
		runErrors = append(runErrors, fmt.Errorf("cleanup expired archives: %w", err))
	} else {
		runSummary.RemovedFiles = len(removedFiles)
		for _, removedFile := range removedFiles {
			r.logger().Printf("retention remove path=%s", removedFile)
		}
	}
	if err := ctx.Err(); err != nil {
		runErrors = append(runErrors, err)
	}

	runSummary.ErrorCount = len(runErrors)
	if len(runErrors) > 0 {
		runErr = errors.Join(runErrors...)
		return
	}

	r.logger().Printf("job finish date=%s at=%q galleries=%d queued=%d", day, runSummary.StartedAtText(), plan.SearchResultsCount, len(plan.Queued))
	return
}

func (r *Runner) processGallery(ctx context.Context, storyArc string, queued QueuedGallery) error {
	gallery := queued.Gallery
	fileName := storage.GalleryFileName(gallery)
	finalPath := storage.FinalGalleryPath(r.Config.LibraryPath(), fileName)
	if _, err := os.Stat(finalPath); err == nil {
		r.logger().Printf("skip existing gallery_id=%d path=%s", gallery.ID, finalPath)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	tempPath := storage.TempArchivePath(finalPath, gallery.ID)
	if err := storage.EnsureDir(filepath.Dir(tempPath)); err != nil {
		return err
	}
	defer os.Remove(tempPath)

	downstreamPath := tempPath + ".official"
	defer os.Remove(downstreamPath)

	download, err := r.downloadGallery(ctx, gallery.ID)
	if err != nil {
		return err
	}
	if err := r.DownloadClient.DownloadToFile(ctx, download.URL, downstreamPath); err != nil {
		return err
	}
	if r.Config.KomgaReadList {
		if err := archive.RewriteCBZ(downstreamPath, tempPath, storyArc, queued.Rank); err != nil {
			return err
		}
	} else if err := os.Rename(downstreamPath, tempPath); err != nil {
		return err
	}

	if err := storage.AtomicReplace(tempPath, finalPath); err != nil {
		return err
	}

	r.logger().Printf("gallery archive ok gallery_id=%d path=%s", gallery.ID, finalPath)
	return nil
}

func (r *Runner) downloadGallery(ctx context.Context, galleryID int64) (nhentai.DownloadResponse, error) {
	if err := r.waitForDownloadIssueSlot(ctx); err != nil {
		return nhentai.DownloadResponse{}, err
	}
	return r.Client.DownloadGallery(ctx, galleryID)
}

func (r *Runner) waitForDownloadIssueSlot(ctx context.Context) error {
	interval := r.Config.DownloadIssueInterval
	if interval <= 0 {
		return nil
	}

	r.downloadIssueMu.Lock()
	defer r.downloadIssueMu.Unlock()

	if !r.lastDownloadIssueAt.IsZero() {
		next := r.lastDownloadIssueAt.Add(interval)
		if delay := time.Until(next); delay > 0 {
			if err := sleepContext(ctx, delay); err != nil {
				return err
			}
		}
	}
	r.lastDownloadIssueAt = time.Now()
	return nil
}

func (r *Runner) logger() *log.Logger {
	if r.Logger != nil {
		return r.Logger
	}
	return log.Default()
}

func (r *Runner) now() time.Time {
	if r.NowFunc != nil {
		return r.NowFunc()
	}
	return time.Now()
}

func (r *Runner) mode() string {
	if r.Mode != "" {
		return r.Mode
	}
	return r.Config.RunMode
}

func (r *Runner) finalizeRun(ctx context.Context, result summary.Result) {
	r.logger().Print(result.LogLine())
	if r.Notifier == nil {
		return
	}
	if err := r.Notifier.Send(ctx, result); err != nil {
		r.logger().Printf("notify bark failed error=%v", err)
	}
}

func formatStoryArc(day string) string {
	return day
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
