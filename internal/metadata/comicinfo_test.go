package metadata

import (
	"strings"
	"testing"

	"nfetcher/internal/nhentai"
)

func TestMarshalComicInfoUsesKomgaFriendlyMetadata(t *testing.T) {
	gallery := nhentai.Gallery{
		ID: 641153,
		Title: nhentai.GalleryTitle{
			Japanese: "テスト作品",
			English:  "Test Gallery",
		},
		Tags: []nhentai.Tag{
			{Name: "tag-a"},
			{Name: "tag-b"},
		},
	}

	data, err := MarshalComicInfo(BuildComicInfo(gallery, "nhentai-popular | 2026-04-04", 3))
	if err != nil {
		t.Fatalf("marshal comic info: %v", err)
	}

	xmlText := string(data)

	if strings.Contains(xmlText, "<Number>") {
		t.Fatalf("expected ComicInfo to omit Number, got %s", xmlText)
	}

	if !strings.Contains(xmlText, "<Title>テスト作品</Title>") {
		t.Fatalf("expected Title in ComicInfo, got %s", xmlText)
	}

	if strings.Contains(xmlText, "<Series>") {
		t.Fatalf("expected ComicInfo to omit Series, got %s", xmlText)
	}

	if !strings.Contains(xmlText, "<StoryArc>nhentai-popular | 2026-04-04</StoryArc>") {
		t.Fatalf("expected source-prefixed StoryArc in ComicInfo, got %s", xmlText)
	}

	if !strings.Contains(xmlText, "<StoryArcNumber>3</StoryArcNumber>") {
		t.Fatalf("expected StoryArcNumber in ComicInfo, got %s", xmlText)
	}
}
