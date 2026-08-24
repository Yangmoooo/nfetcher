package job

import (
	"context"
	"fmt"

	"nfetcher/internal/config"
	"nfetcher/internal/nhentai"
	"nfetcher/internal/storage"
)

type DuplicateGallery struct {
	QueuedGallery
	ExistingPath string
}

type PlanResult struct {
	SearchResultsCount int
	Duplicates         []DuplicateGallery
	Queued             []QueuedGallery
}

type PlanOptions struct {
	Log                  bool
	ExistingGalleryPaths map[int64]string
}

func (r *Runner) BuildPlan(ctx context.Context, options PlanOptions) (PlanResult, error) {
	existingGalleryPaths := options.ExistingGalleryPaths
	if existingGalleryPaths == nil {
		var err error
		existingGalleryPaths, err = storage.ExistingGalleryPaths(config.LibraryDirPath)
		if err != nil {
			return PlanResult{}, fmt.Errorf("scan existing library: %w", err)
		}
	}

	searchResult, err := r.Client.Search(ctx, r.Config.SearchQuery, r.Config.SearchSort, r.Config.SearchPage)
	if err != nil {
		return PlanResult{}, fmt.Errorf("search galleries: %w", err)
	}

	if options.Log {
		r.logger().Printf("search results count=%d", len(searchResult.Result))
	}

	plan := PlanResult{
		SearchResultsCount: len(searchResult.Result),
		Queued:             make([]QueuedGallery, 0, len(searchResult.Result)),
		Duplicates:         make([]DuplicateGallery, 0),
	}

	scheduledGalleryIDs := make(map[int64]struct{}, len(searchResult.Result))
	for rank, item := range searchResult.Result {
		if err := ctx.Err(); err != nil {
			return PlanResult{}, err
		}

		gallery := nhentai.Gallery{
			ID:       item.ID,
			MediaID:  item.MediaID,
			Title:    item.Title,
			NumPages: item.NumPages,
		}
		queued := QueuedGallery{
			Gallery: gallery,
			Rank:    rank + 1,
		}

		if options.Log {
			r.logger().Printf("gallery search item gallery_id=%d pages=%d", gallery.ID, gallery.NumPages)
		}

		if existingPath, exists := existingGalleryPaths[gallery.ID]; exists {
			plan.Duplicates = append(plan.Duplicates, DuplicateGallery{
				QueuedGallery: queued,
				ExistingPath:  existingPath,
			})
			if options.Log {
				r.logger().Printf("skip duplicate gallery_id=%d existing_path=%s", gallery.ID, existingPath)
			}
			continue
		}

		if _, exists := scheduledGalleryIDs[gallery.ID]; exists {
			if options.Log {
				r.logger().Printf("skip duplicate in-run gallery_id=%d", gallery.ID)
			}
			continue
		}

		scheduledGalleryIDs[gallery.ID] = struct{}{}
		plan.Queued = append(plan.Queued, queued)
	}

	SortQueuedGalleriesByPageCountDesc(plan.Queued)
	if options.Log && len(plan.Queued) > 0 {
		smallestPages := plan.Queued[len(plan.Queued)-1].Gallery.NumPages
		largestPages := plan.Queued[0].Gallery.NumPages
		r.logger().Printf(
			"gallery queue count=%d gallery_concurrency=%d order=pages-desc largest_pages=%d smallest_pages=%d",
			len(plan.Queued),
			r.Config.GalleryConcurrency,
			largestPages,
			smallestPages,
		)
	}

	return plan, nil
}
