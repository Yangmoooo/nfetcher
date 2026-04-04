package metadata

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"nfetcher/internal/nhentai"
	"nfetcher/internal/storage"
)

const adultsOnlyAgeRating = "Adults Only 18+"

type ComicInfo struct {
	XMLName        xml.Name `xml:"ComicInfo"`
	Title          string   `xml:"Title,omitempty"`
	StoryArc       string   `xml:"StoryArc,omitempty"`
	StoryArcNumber string   `xml:"StoryArcNumber,omitempty"`
	Web            string   `xml:"Web,omitempty"`
	Tags           string   `xml:"Tags,omitempty"`
	AgeRating      string   `xml:"AgeRating,omitempty"`
}

func BuildComicInfo(gallery nhentai.Gallery, storyArc string, rank int) ComicInfo {
	title := storage.ChooseTitle(gallery)
	return ComicInfo{
		Title:          title,
		StoryArc:       storyArc,
		StoryArcNumber: strconv.Itoa(rank),
		Web:            fmt.Sprintf("https://nhentai.net/g/%d/", gallery.ID),
		Tags:           joinTagNames(gallery.Tags),
		AgeRating:      adultsOnlyAgeRating,
	}
}

func MarshalComicInfo(info ComicInfo) ([]byte, error) {
	payload, err := xml.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, err
	}

	data := append([]byte(xml.Header), payload...)
	data = append(data, '\n')
	return data, nil
}

func joinTagNames(tags []nhentai.Tag) string {
	if len(tags) == 0 {
		return ""
	}

	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	return strings.Join(names, ",")
}
