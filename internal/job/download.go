package job

import (
	"context"
	"sort"
	"sync"

	"nfetcher/internal/nhentai"
)

type FileDownloader interface {
	DownloadToFile(ctx context.Context, rawURL, dstPath string) error
}

type GalleryResult struct {
	ID      int64
	Gallery nhentai.Gallery
	Err     error
}

type QueuedGallery struct {
	Gallery nhentai.Gallery
	Rank    int
}

type GalleryProcessResult struct {
	QueuedGallery QueuedGallery
	Err           error
}

func FetchDetails(ctx context.Context, client *nhentai.Client, ids []int64, workers int) <-chan GalleryResult {
	jobs := make(chan int64)
	results := make(chan GalleryResult)

	var workersWG sync.WaitGroup
	for workerIndex := 0; workerIndex < workers; workerIndex++ {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for id := range jobs {
				gallery, err := client.GetGallery(ctx, id)
				select {
				case <-ctx.Done():
					return
				case results <- GalleryResult{ID: id, Gallery: gallery, Err: err}:
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, id := range ids {
			select {
			case <-ctx.Done():
				return
			case jobs <- id:
			}
		}
	}()

	go func() {
		workersWG.Wait()
		close(results)
	}()

	return results
}

func SortQueuedGalleriesByPageCountDesc(galleries []QueuedGallery) {
	sort.Slice(galleries, func(i, j int) bool {
		leftPages := galleries[i].Gallery.NumPages
		rightPages := galleries[j].Gallery.NumPages
		if leftPages != rightPages {
			return leftPages > rightPages
		}
		return galleries[i].Gallery.ID < galleries[j].Gallery.ID
	})
}

func ProcessGalleries(ctx context.Context, galleries []QueuedGallery, workers int, process func(context.Context, QueuedGallery) error) <-chan GalleryProcessResult {
	jobs := make(chan QueuedGallery)
	results := make(chan GalleryProcessResult)

	var workersWG sync.WaitGroup
	for workerIndex := 0; workerIndex < workers; workerIndex++ {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for gallery := range jobs {
				err := process(ctx, gallery)
				select {
				case <-ctx.Done():
					return
				case results <- GalleryProcessResult{QueuedGallery: gallery, Err: err}:
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, gallery := range galleries {
			select {
			case <-ctx.Done():
				return
			case jobs <- gallery:
			}
		}
	}()

	go func() {
		workersWG.Wait()
		close(results)
	}()

	return results
}
