package storage

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"nfetcher/internal/nhentai"
)

const maxFileNameBytes = 255

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
	available := maxFileNameBytes - len([]byte(suffix))
	if available < 1 {
		return id + ".cbz"
	}

	title = truncateUTF8Bytes(title, available)
	return title + suffix
}

func GalleryDirName(gallery nhentai.Gallery) string {
	fileName := GalleryFileName(gallery)
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func truncateUTF8Bytes(value string, limit int) string {
	if len([]byte(value)) <= limit {
		return value
	}

	var builder strings.Builder
	builder.Grow(limit)
	for _, runeValue := range value {
		runeBytes := utf8.RuneLen(runeValue)
		if runeBytes < 0 || builder.Len()+runeBytes > limit {
			break
		}
		builder.WriteRune(runeValue)
	}
	return strings.TrimSpace(builder.String())
}
