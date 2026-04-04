package job

import (
	"context"
	"fmt"

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
	Errors             []error
	DetailFailedIDs    []int64
}

type PlanOptions struct {
	Log bool
}

func (r *Runner) BuildPlan(ctx context.Context, options PlanOptions) (PlanResult, error) {
	existingGalleryPaths, err := storage.ExistingGalleryPaths(r.Config.LibraryDir)
	if err != nil {
		return PlanResult{}, fmt.Errorf("scan existing library: %w", err)
	}

	searchResult, err := r.Client.Search(ctx, r.Config.SearchQuery, r.Config.SearchSort, r.Config.SearchPage)
	if err != nil {
		return PlanResult{}, fmt.Errorf("search galleries: %w", err)
	}

	ids := make([]int64, 0, len(searchResult.Result))
	searchRanks := make(map[int64]int, len(searchResult.Result))
	for _, item := range searchResult.Result {
		ids = append(ids, item.ID)
		searchRanks[item.ID] = len(ids)
	}

	if options.Log {
		r.logger().Printf("search results count=%d", len(ids))
	}

	plan := PlanResult{
		SearchResultsCount: len(ids),
		Queued:             make([]QueuedGallery, 0, len(ids)),
		Duplicates:         make([]DuplicateGallery, 0),
		Errors:             make([]error, 0),
		DetailFailedIDs:    make([]int64, 0),
	}

	scheduledGalleryIDs := make(map[int64]struct{}, len(ids))
	for result := range FetchDetails(ctx, r.Client, ids, r.Config.DetailConcurrency) {
		if result.Err != nil {
			plan.Errors = append(plan.Errors, fmt.Errorf("gallery %d: %w", result.ID, result.Err))
			plan.DetailFailedIDs = append(plan.DetailFailedIDs, result.ID)
			if options.Log {
				r.logger().Printf("gallery detail failed gallery_id=%d error=%v", result.ID, result.Err)
			}
			continue
		}

		rank := searchRanks[result.Gallery.ID]
		queued := QueuedGallery{
			Gallery: result.Gallery,
			Rank:    rank,
		}

		if options.Log {
			r.logger().Printf("gallery detail ok gallery_id=%d pages=%d", result.Gallery.ID, len(result.Gallery.Pages))
		}

		if existingPath, exists := existingGalleryPaths[result.Gallery.ID]; exists {
			plan.Duplicates = append(plan.Duplicates, DuplicateGallery{
				QueuedGallery: queued,
				ExistingPath:  existingPath,
			})
			if options.Log {
				r.logger().Printf("skip duplicate gallery_id=%d existing_path=%s", result.Gallery.ID, existingPath)
			}
			continue
		}

		if _, exists := scheduledGalleryIDs[result.Gallery.ID]; exists {
			if options.Log {
				r.logger().Printf("skip duplicate in-run gallery_id=%d", result.Gallery.ID)
			}
			continue
		}

		scheduledGalleryIDs[result.Gallery.ID] = struct{}{}
		plan.Queued = append(plan.Queued, queued)
	}

	SortQueuedGalleriesByPageCountDesc(plan.Queued)
	if options.Log && len(plan.Queued) > 0 {
		smallestPages := len(plan.Queued[len(plan.Queued)-1].Gallery.Pages)
		largestPages := len(plan.Queued[0].Gallery.Pages)
		r.logger().Printf(
			"gallery queue count=%d gallery_concurrency=%d page_concurrency=%d order=pages-desc largest_pages=%d smallest_pages=%d",
			len(plan.Queued),
			r.Config.GalleryConcurrency,
			r.Config.PageConcurrency,
			largestPages,
			smallestPages,
		)
	}

	return plan, nil
}
