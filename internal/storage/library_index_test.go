package storage

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanLibraryIndexUsesFilenameIDAndStoryArcDate(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "Alpha - 641153.cbz")
	writeTestCBZ(t, archivePath, `<?xml version="1.0" encoding="UTF-8"?>
<ComicInfo>
  <StoryArc>nhentai-popular | 2026-04-04</StoryArc>
  <Web>https://nhentai.net/g/641153/</Web>
</ComicInfo>
`)

	index, err := ScanLibraryIndex(root)
	if err != nil {
		t.Fatalf("scan library index: %v", err)
	}

	if len(index.Archives) != 1 {
		t.Fatalf("expected 1 archive, got %#v", index.Archives)
	}

	archive := index.Archives[0]
	if archive.GalleryID != 641153 {
		t.Fatalf("expected gallery id 641153, got %#v", archive)
	}
	if archive.FetchedDate != "2026-04-04" {
		t.Fatalf("expected fetched date 2026-04-04, got %#v", archive)
	}

	paths := index.ExistingGalleryPaths()
	if paths[641153] != archivePath {
		t.Fatalf("expected indexed path %q, got %#v", archivePath, paths)
	}
}

func TestScanLibraryIndexFallsBackToComicInfoWeb(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "No ID Here.cbz")
	writeTestCBZ(t, archivePath, `<?xml version="1.0" encoding="UTF-8"?>
<ComicInfo>
  <StoryArc>2026-04-05</StoryArc>
  <Web>https://nhentai.net/g/641154/</Web>
</ComicInfo>
`)

	index, err := ScanLibraryIndex(root)
	if err != nil {
		t.Fatalf("scan library index: %v", err)
	}

	paths := index.ExistingGalleryPaths()
	if paths[641154] != archivePath {
		t.Fatalf("expected ComicInfo web fallback path %q, got %#v", archivePath, paths)
	}
}

func TestCleanupExpiredFromIndex(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "Old - 641155.cbz")
	keepPath := filepath.Join(root, "Keep - 641156.cbz")
	unknownPath := filepath.Join(root, "Unknown - 641157.cbz")

	writeTestCBZ(t, oldPath, `<?xml version="1.0" encoding="UTF-8"?>
<ComicInfo>
  <StoryArc>nhentai-popular | 2026-04-01</StoryArc>
  <Web>https://nhentai.net/g/641155/</Web>
</ComicInfo>
`)
	writeTestCBZ(t, keepPath, `<?xml version="1.0" encoding="UTF-8"?>
<ComicInfo>
  <StoryArc>popular | 2026-04-08</StoryArc>
  <Web>https://nhentai.net/g/641156/</Web>
</ComicInfo>
`)
	writeTestCBZ(t, unknownPath, `<?xml version="1.0" encoding="UTF-8"?>
<ComicInfo>
  <Web>https://nhentai.net/g/641157/</Web>
</ComicInfo>
`)

	index, err := ScanLibraryIndex(root)
	if err != nil {
		t.Fatalf("scan library index: %v", err)
	}

	removed, err := index.RemoveExpired(time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC), 7)
	if err != nil {
		t.Fatalf("remove expired: %v", err)
	}

	if len(removed) != 1 || removed[0] != oldPath {
		t.Fatalf("expected only old archive removed, got %#v", removed)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old archive deleted, stat err=%v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("expected recent archive kept, stat err=%v", err)
	}
	if _, err := os.Stat(unknownPath); err != nil {
		t.Fatalf("expected archive without date kept, stat err=%v", err)
	}
}

func writeTestCBZ(t *testing.T, path, comicInfo string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create cbz: %v", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)

	pageWriter, err := zipWriter.Create("001.jpg")
	if err != nil {
		t.Fatalf("create page entry: %v", err)
	}
	if _, err := pageWriter.Write([]byte("page")); err != nil {
		t.Fatalf("write page entry: %v", err)
	}

	infoWriter, err := zipWriter.Create("ComicInfo.xml")
	if err != nil {
		t.Fatalf("create ComicInfo entry: %v", err)
	}
	if _, err := infoWriter.Write([]byte(comicInfo)); err != nil {
		t.Fatalf("write ComicInfo entry: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close cbz zip: %v", err)
	}
}
