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
	"nfetcher/internal/storage"
)

type Runner struct {
	Config      config.Config
	Client      *nhentai.Client
	ImageClient Downloader
	Logger      *log.Logger
	NowFunc     func() time.Time
}

func (r *Runner) Run(ctx context.Context) error {
	now := r.now()
	day := now.Format("2006-01-02")
	r.logger().Printf("job start date=%s query=%q sort=%q page=%d", day, r.Config.SearchQuery, r.Config.SearchSort, r.Config.SearchPage)

	existingGalleryPaths, err := storage.ExistingGalleryPaths(r.Config.LibraryDir)
	if err != nil {
		return fmt.Errorf("scan existing library: %w", err)
	}

	searchResult, err := r.Client.Search(ctx, r.Config.SearchQuery, r.Config.SearchSort, r.Config.SearchPage)
	if err != nil {
		return fmt.Errorf("search galleries: %w", err)
	}

	ids := make([]int64, 0, len(searchResult.Result))
	searchRanks := make(map[int64]int, len(searchResult.Result))
	for _, item := range searchResult.Result {
		ids = append(ids, item.ID)
		searchRanks[item.ID] = len(ids)
	}

	r.logger().Printf("search results count=%d", len(ids))

	var runErrors []error
	galleries := make([]QueuedGallery, 0, len(ids))
	scheduledGalleryIDs := make(map[int64]struct{}, len(ids))
	for result := range FetchDetails(ctx, r.Client, ids, r.Config.DetailConcurrency) {
		if result.Err != nil {
			runErrors = append(runErrors, result.Err)
			r.logger().Printf("gallery detail failed: %v", result.Err)
			continue
		}

		r.logger().Printf("gallery detail ok gallery_id=%d pages=%d", result.Gallery.ID, len(result.Gallery.Pages))
		if existingPath, exists := existingGalleryPaths[result.Gallery.ID]; exists {
			r.logger().Printf("skip duplicate gallery_id=%d existing_path=%s", result.Gallery.ID, existingPath)
			continue
		}

		if _, exists := scheduledGalleryIDs[result.Gallery.ID]; exists {
			r.logger().Printf("skip duplicate in-run gallery_id=%d", result.Gallery.ID)
			continue
		}

		scheduledGalleryIDs[result.Gallery.ID] = struct{}{}
		galleries = append(galleries, QueuedGallery{
			Gallery: result.Gallery,
			Rank:    searchRanks[result.Gallery.ID],
		})
	}

	SortQueuedGalleriesByPageCountDesc(galleries)
	if len(galleries) > 0 {
		smallestPages := len(galleries[len(galleries)-1].Gallery.Pages)
		largestPages := len(galleries[0].Gallery.Pages)
		r.logger().Printf(
			"gallery queue count=%d gallery_concurrency=%d page_concurrency=%d order=pages-desc largest_pages=%d smallest_pages=%d",
			len(galleries),
			r.Config.GalleryConcurrency,
			r.Config.PageConcurrency,
			largestPages,
			smallestPages,
		)
	}

	for result := range ProcessGalleries(ctx, galleries, r.Config.GalleryConcurrency, func(workerCtx context.Context, gallery QueuedGallery) error {
		return r.processGallery(workerCtx, now, gallery)
	}) {
		if result.Err != nil {
			runErrors = append(runErrors, fmt.Errorf("process gallery %d: %w", result.QueuedGallery.Gallery.ID, result.Err))
			r.logger().Printf("gallery archive failed gallery_id=%d error=%v", result.QueuedGallery.Gallery.ID, result.Err)
			continue
		}

		existingGalleryPaths[result.QueuedGallery.Gallery.ID] = storage.FinalGalleryPath(r.Config.LibraryDir, now, storage.GalleryFileName(result.QueuedGallery.Gallery))
	}

	removedDirs, err := CleanupOldDirs(r.Config.LibraryDir, now, r.Config.RetentionDays)
	if err != nil {
		runErrors = append(runErrors, fmt.Errorf("cleanup old directories: %w", err))
	} else {
		for _, removedDir := range removedDirs {
			r.logger().Printf("retention remove dir=%s", removedDir)
		}
	}

	if len(runErrors) > 0 {
		return errors.Join(runErrors...)
	}

	r.logger().Printf("job finish date=%s galleries=%d queued=%d", day, len(ids), len(galleries))
	return nil
}

func (r *Runner) processGallery(ctx context.Context, now time.Time, queued QueuedGallery) error {
	gallery := queued.Gallery
	finalPath := storage.FinalGalleryPath(r.Config.LibraryDir, now, storage.GalleryFileName(gallery))
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

	comicInfo, err := metadata.MarshalComicInfo(metadata.BuildComicInfo(gallery, queued.Rank))
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
