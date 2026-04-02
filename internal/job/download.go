package job

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	"nfetcher/internal/nhentai"
)

type Downloader interface {
	DownloadToFile(ctx context.Context, rawURL, dstPath string) error
}

type GalleryResult struct {
	Gallery nhentai.Gallery
	Err     error
}

type GalleryProcessResult struct {
	Gallery nhentai.Gallery
	Err     error
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
				case results <- GalleryResult{Gallery: gallery, Err: err}:
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

func SortGalleriesByPageCountDesc(galleries []nhentai.Gallery) {
	sort.Slice(galleries, func(i, j int) bool {
		leftPages := len(galleries[i].Pages)
		rightPages := len(galleries[j].Pages)
		if leftPages != rightPages {
			return leftPages > rightPages
		}
		return galleries[i].ID < galleries[j].ID
	})
}

func ProcessGalleries(ctx context.Context, galleries []nhentai.Gallery, workers int, process func(context.Context, nhentai.Gallery) error) <-chan GalleryProcessResult {
	jobs := make(chan nhentai.Gallery)
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
				case results <- GalleryProcessResult{Gallery: gallery, Err: err}:
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

func pageFileName(number int, pagePath string) string {
	return fmt.Sprintf("%03d%s", number, filepath.Ext(pagePath))
}

func downloadPages(ctx context.Context, apiClient *nhentai.Client, imageClient Downloader, gallery nhentai.Gallery, stageDir string, workers int) error {
	if len(gallery.Pages) == 0 {
		return fmt.Errorf("gallery %d has no pages", gallery.ID)
	}

	pageCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan nhentai.Page)
	errCh := make(chan error, 1)

	var workersWG sync.WaitGroup
	for workerIndex := 0; workerIndex < workers; workerIndex++ {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for {
				select {
				case <-pageCtx.Done():
					return
				case page, ok := <-jobs:
					if !ok {
						return
					}

					dstPath := filepath.Join(stageDir, pageFileName(page.Number, page.Path))
					rawURL := apiClient.ImageURL(page.Path)
					if err := imageClient.DownloadToFile(pageCtx, rawURL, dstPath); err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, page := range gallery.Pages {
			select {
			case <-pageCtx.Done():
				return
			case jobs <- page:
			}
		}
	}()

	workersWG.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
