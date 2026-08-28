package job

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nfetcher/internal/config"
	"nfetcher/internal/nhentai"
)

type staticDownloadAPI struct{}

func (staticDownloadAPI) Do(context.Context, *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"url":"https://download.example/test.cbz","expires_at":1}`)),
	}, nil
}

type fixtureDownloader struct {
	data []byte
}

func (d fixtureDownloader) DownloadToFile(_ context.Context, _, dstPath string) error {
	return os.WriteFile(dstPath, d.data, 0o600)
}

func TestWaitForDownloadIssueSlotHonorsConfiguredInterval(t *testing.T) {
	runner := &Runner{
		Config: config.Config{
			DownloadIssueInterval: 20 * time.Millisecond,
		},
	}

	if err := runner.waitForDownloadIssueSlot(context.Background()); err != nil {
		t.Fatalf("first wait: %v", err)
	}

	startedAt := time.Now()
	if err := runner.waitForDownloadIssueSlot(context.Background()); err != nil {
		t.Fatalf("second wait: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed < runner.Config.DownloadIssueInterval {
		t.Fatalf("expected second wait to honor interval, elapsed %s", elapsed)
	}
}

func TestWaitForDownloadIssueSlotCanBeDisabled(t *testing.T) {
	runner := &Runner{
		Config: config.Config{
			DownloadIssueInterval: 0,
		},
	}

	startedAt := time.Now()
	if err := runner.waitForDownloadIssueSlot(context.Background()); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 10*time.Millisecond {
		t.Fatalf("expected disabled interval to return immediately, elapsed %s", elapsed)
	}
}

func TestProcessGalleryOnlyAddsKomgaReadListMetadataWhenEnabled(t *testing.T) {
	for _, test := range []struct {
		name                string
		komgaReadList       bool
		wantStoryArc        string
		wantArcNumber       string
		wantAlternateSeries bool
	}{
		{name: "generic", wantStoryArc: "official-story-arc", wantArcNumber: "9", wantAlternateSeries: true},
		{name: "komga", komgaReadList: true, wantStoryArc: "2026-08-27", wantArcNumber: "4"},
	} {
		t.Run(test.name, func(t *testing.T) {
			libraryDir := t.TempDir()
			runner := &Runner{
				Config: config.Config{
					LibraryDir:            libraryDir,
					KomgaReadList:         test.komgaReadList,
					DownloadIssueInterval: 0,
				},
				Client:         nhentai.NewClient(staticDownloadAPI{}),
				DownloadClient: fixtureDownloader{data: testCBZFixture(t)},
			}

			queued := QueuedGallery{
				Gallery: nhentai.Gallery{
					ID: 645649,
					Title: nhentai.GalleryTitle{
						English: "Test title",
					},
				},
				Rank: 4,
			}
			if err := runner.processGallery(context.Background(), "2026-08-27", queued); err != nil {
				t.Fatalf("process gallery: %v", err)
			}

			finalPath := filepath.Join(libraryDir, "Test title - 645649.cbz")
			comicInfo := readComicInfoFromCBZ(t, finalPath)
			if !strings.Contains(comicInfo, "<StoryArc>"+test.wantStoryArc+"</StoryArc>") {
				t.Fatalf("expected StoryArc %q, got %s", test.wantStoryArc, comicInfo)
			}
			if !strings.Contains(comicInfo, "<StoryArcNumber>"+test.wantArcNumber+"</StoryArcNumber>") {
				t.Fatalf("expected StoryArcNumber %q, got %s", test.wantArcNumber, comicInfo)
			}
			hasAlternateSeries := strings.Contains(comicInfo, "<AlternateSeries>")
			if hasAlternateSeries != test.wantAlternateSeries {
				t.Fatalf("expected AlternateSeries present=%t, got %s", test.wantAlternateSeries, comicInfo)
			}
		})
	}
}

func testCBZFixture(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	comicInfo, err := writer.Create("ComicInfo.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := comicInfo.Write([]byte(`<ComicInfo><StoryArc>official-story-arc</StoryArc><StoryArcNumber>9</StoryArcNumber><AlternateSeries>Official Title</AlternateSeries></ComicInfo>`)); err != nil {
		t.Fatal(err)
	}
	page, err := writer.Create("001.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := page.Write([]byte("page")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func readComicInfoFromCBZ(t *testing.T, path string) string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != "ComicInfo.xml" {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	t.Fatal("ComicInfo.xml not found")
	return ""
}
