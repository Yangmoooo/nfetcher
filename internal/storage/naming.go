package storage

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"nfetcher/internal/nhentai"
)

const maxFileNameRunes = 200

var invalidChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)
var multiSpace = regexp.MustCompile(`\s+`)

func ChooseTitle(gallery nhentai.Gallery) string {
	if strings.TrimSpace(gallery.Title.Japanese) != "" {
		return gallery.Title.Japanese
	}
	if strings.TrimSpace(gallery.Title.English) != "" {
		return gallery.Title.English
	}
	return strconv.FormatInt(gallery.ID, 10)
}

func SanitizeFileName(raw string) string {
	name := invalidChars.ReplaceAllString(strings.TrimSpace(raw), "_")
	name = multiSpace.ReplaceAllString(name, " ")
	name = strings.Trim(name, " .")
	if name == "" {
		return "untitled"
	}
	return name
}

func GalleryFileName(gallery nhentai.Gallery) string {
	id := strconv.FormatInt(gallery.ID, 10)
	title := SanitizeFileName(ChooseTitle(gallery))
	if title == id {
		return id + ".cbz"
	}

	suffix := fmt.Sprintf(" - %s.cbz", id)
	available := maxFileNameRunes - len([]rune(suffix))
	if available < 1 {
		return id + ".cbz"
	}

	title = truncateRunes(title, available)
	return title + suffix
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit]))
}
