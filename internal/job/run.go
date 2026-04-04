package job

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"nfetcher/internal/archive"
	"nfetcher/internal/config"
	"nfetcher/internal/metadata"
	"nfetcher/internal/nhentai"
	"nfetcher/internal/notify"
	"nfetcher/internal/storage"
	"nfetcher/internal/summary"
)

type Runner struct {
	Config      config.Config
	Client      *nhentai.Client
	ImageClient Downloader
	Logger      *log.Logger
	NowFunc     func() time.Time
	Mode        string
	Notifier    notify.Sender
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

	index, err := storage.ScanLibraryIndex(config.LibraryDirPath)
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
	runSummary.DetailErrors = len(plan.Errors)
	runSummary.FailedGalleryIDs = append(runSummary.FailedGalleryIDs, plan.DetailFailedIDs...)

	runErrors := append([]error(nil), plan.Errors...)
	for processResult := range ProcessGalleries(ctx, plan.Queued, r.Config.GalleryConcurrency, func(workerCtx context.Context, gallery QueuedGallery) error {
		return r.processGallery(workerCtx, now, storyArc, gallery)
	}) {
		if processResult.Err != nil {
			runErrors = append(runErrors, fmt.Errorf("process gallery %d: %w", processResult.QueuedGallery.Gallery.ID, processResult.Err))
			r.logger().Printf("gallery archive failed gallery_id=%d error=%v", processResult.QueuedGallery.Gallery.ID, processResult.Err)
			runSummary.ArchivedFailed++
			runSummary.FailedGalleryIDs = append(runSummary.FailedGalleryIDs, processResult.QueuedGallery.Gallery.ID)
			continue
		}

		runSummary.ArchivedOK++
		finalPath := storage.FinalGalleryPath(config.LibraryDirPath, storage.GalleryFileName(processResult.QueuedGallery.Gallery))
		index.Archives = append(index.Archives, storage.LibraryArchive{
			Path:        finalPath,
			GalleryID:   processResult.QueuedGallery.Gallery.ID,
			FetchedDate: day,
		})
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

	runSummary.ErrorCount = len(runErrors)
	if len(runErrors) > 0 {
		runErr = errors.Join(runErrors...)
		return
	}

	r.logger().Printf("job finish date=%s at=%q galleries=%d queued=%d", day, runSummary.StartedAtText(), plan.SearchResultsCount, len(plan.Queued))
	return
}

func (r *Runner) processGallery(ctx context.Context, now time.Time, storyArc string, queued QueuedGallery) error {
	gallery := queued.Gallery
	fileName := storage.GalleryFileName(gallery)
	finalPath := storage.FinalGalleryPath(config.LibraryDirPath, fileName)
	if _, err := os.Stat(finalPath); err == nil {
		r.logger().Printf("skip existing gallery_id=%d path=%s", gallery.ID, finalPath)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	stageDir := storage.StageDir(now, gallery.ID)
	if err := storage.EnsureDir(stageDir); err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := downloadPages(ctx, r.Client, r.ImageClient, gallery, stageDir, r.Config.PageConcurrency); err != nil {
		return err
	}

	tempPath := storage.TempArchivePath(finalPath, gallery.ID)
	if err := storage.EnsureDir(filepath.Dir(tempPath)); err != nil {
		return err
	}
	defer os.Remove(tempPath)

	comicInfo, err := metadata.MarshalComicInfo(metadata.BuildComicInfo(gallery, storyArc, queued.Rank))
	if err != nil {
		return err
	}

	if err := archive.WriteCBZ(stageDir, tempPath, []archive.ExtraFile{
		{Name: "ComicInfo.xml", Data: comicInfo},
	}); err != nil {
		return err
	}

	if err := storage.AtomicReplace(tempPath, finalPath); err != nil {
		return err
	}

	r.logger().Printf("gallery archive ok gallery_id=%d path=%s", gallery.ID, finalPath)
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
	return "nhentai-popular | " + day
}
