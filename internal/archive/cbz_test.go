package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteCBZPatchesComicInfoAndKeepsEntries(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "official.cbz")
	dstPath := filepath.Join(t.TempDir(), "patched.cbz")
	writeTestCBZ(t, srcPath, map[string]string{
		"ComicInfo.xml": `<?xml version="1.0" encoding="utf-8"?>
<ComicInfo>
  <Title>Official Title</Title>
</ComicInfo>
`,
		"001.jpg": "image-data",
	})

	if err := RewriteCBZ(srcPath, dstPath, "2026-05-06", 2); err != nil {
		t.Fatalf("rewrite cbz: %v", err)
	}

	entries := readTestCBZ(t, dstPath)
	if entries["001.jpg"] != "image-data" {
		t.Fatalf("expected image entry to be preserved, got %#v", entries)
	}
	comicInfo := entries["ComicInfo.xml"]
	if !strings.Contains(comicInfo, "<Title>Official Title</Title>") {
		t.Fatalf("expected official ComicInfo fields to be preserved, got %s", comicInfo)
	}
	if !strings.Contains(comicInfo, "<StoryArc>2026-05-06</StoryArc>") {
		t.Fatalf("expected StoryArc patch, got %s", comicInfo)
	}
	if !strings.Contains(comicInfo, "<StoryArcNumber>2</StoryArcNumber>") {
		t.Fatalf("expected StoryArcNumber patch, got %s", comicInfo)
	}
}

func TestRewriteCBZRequiresComicInfo(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "official.cbz")
	dstPath := filepath.Join(t.TempDir(), "patched.cbz")
	writeTestCBZ(t, srcPath, map[string]string{"001.jpg": "image-data"})

	err := RewriteCBZ(srcPath, dstPath, "2026-05-06", 2)
	if err == nil || !strings.Contains(err.Error(), "ComicInfo.xml not found") {
		t.Fatalf("expected missing ComicInfo error, got %v", err)
	}
}

func writeTestCBZ(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()
	for name, data := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
}

func readTestCBZ(t *testing.T, path string) map[string]string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(entry)
		entry.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = string(data)
	}
	return entries
}
