package metadata

import (
	"strings"
	"testing"

	"nfetcher/internal/nhentai"
)

func TestMarshalComicInfoOmitsChapterFieldsForKavita(t *testing.T) {
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

	data, err := MarshalComicInfo(BuildComicInfo(gallery, "2026-04-04", 3))
	if err != nil {
		t.Fatalf("marshal comic info: %v", err)
	}

	xmlText := string(data)

	if strings.Contains(xmlText, "<Title>") {
		t.Fatalf("expected ComicInfo to omit Title, got %s", xmlText)
	}

	if strings.Contains(xmlText, "<Number>") {
		t.Fatalf("expected ComicInfo to omit Number, got %s", xmlText)
	}

	if !strings.Contains(xmlText, "<Series>テスト作品</Series>") {
		t.Fatalf("expected Series in ComicInfo, got %s", xmlText)
	}

	if !strings.Contains(xmlText, "<StoryArc>2026-04-04</StoryArc>") {
		t.Fatalf("expected StoryArc in ComicInfo, got %s", xmlText)
	}

	if !strings.Contains(xmlText, "<StoryArcNumber>3</StoryArcNumber>") {
		t.Fatalf("expected StoryArcNumber in ComicInfo, got %s", xmlText)
	}
}
