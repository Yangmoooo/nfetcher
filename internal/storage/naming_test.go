package storage

import (
	"strings"
	"testing"

	"nfetcher/internal/nhentai"
)

func TestGalleryFileNameFitsFilesystemComponentLimitForMultibyteTitles(t *testing.T) {
	title := strings.Repeat("漫", 200)
	name := GalleryFileName(nhentai.Gallery{
		ID: 645649,
		Title: nhentai.GalleryTitle{
			Japanese: title,
		},
	})

	if got := len([]byte(name)); got > 255 {
		t.Fatalf("gallery filename is %d bytes, want <= 255: %q", got, name)
	}
}
