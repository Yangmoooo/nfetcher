package metadata

import (
	"strings"
	"testing"
)

func TestPatchComicInfoAddsStoryArcFields(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ComicInfo xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <Title>Official Title</Title>
  <Series>original</Series>
  <Translator>todaya</Translator>
  <Tags>big breasts, sole female</Tags>
</ComicInfo>
`)

	data, err := PatchComicInfo(input, "nhentai-popular | 2026-05-06", 7)
	if err != nil {
		t.Fatalf("patch comic info: %v", err)
	}

	text := string(data)
	for _, expected := range []string{
		`<ComicInfo xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">`,
		"<Title>Official Title</Title>",
		"<Series>original</Series>",
		"<Translator>todaya</Translator>",
		"<Tags>big breasts, sole female</Tags>",
		"<StoryArc>nhentai-popular | 2026-05-06</StoryArc>",
		"<StoryArcNumber>7</StoryArcNumber>",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in patched ComicInfo, got %s", expected, text)
		}
	}
}

func TestPatchComicInfoReplacesStoryArcFields(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ComicInfo>
  <Title>Official Title</Title>
  <StoryArc>old</StoryArc>
  <StoryArcNumber>99</StoryArcNumber>
</ComicInfo>
`)

	data, err := PatchComicInfo(input, "nhentai-popular | 2026-05-06", 3)
	if err != nil {
		t.Fatalf("patch comic info: %v", err)
	}

	text := string(data)
	if strings.Contains(text, "<StoryArc>old</StoryArc>") || strings.Contains(text, "<StoryArcNumber>99</StoryArcNumber>") {
		t.Fatalf("expected old StoryArc fields to be replaced, got %s", text)
	}
	if !strings.Contains(text, "<StoryArc>nhentai-popular | 2026-05-06</StoryArc>") {
		t.Fatalf("expected StoryArc replacement, got %s", text)
	}
	if !strings.Contains(text, "<StoryArcNumber>3</StoryArcNumber>") {
		t.Fatalf("expected StoryArcNumber replacement, got %s", text)
	}
}
