package storage

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const fetchedDateLayout = "2006-01-02"

type LibraryIndex struct {
	Archives []LibraryArchive
}

type LibraryArchive struct {
	Path        string
	GalleryID   int64
	FetchedDate string
}

type archiveComicInfo struct {
	Web string `xml:"Web"`
}

func ScanLibraryIndex(root string) (LibraryIndex, error) {
	index := LibraryIndex{
		Archives: []LibraryArchive{},
	}

	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return index, nil
		}
		return LibraryIndex{}, err
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			switch entry.Name() {
			case ".tmp", ".nfetcher":
				return filepath.SkipDir
			default:
				return nil
			}
		}

		if filepath.Ext(path) != ".cbz" {
			return nil
		}

		archive := LibraryArchive{Path: path}
		fileInfo, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		archive.FetchedDate = fileInfo.ModTime().Format(fetchedDateLayout)
		if galleryID, ok := GalleryIDFromPath(path); ok {
			archive.GalleryID = galleryID
		}

		comicInfo, _ := readArchiveComicInfo(path)
		if archive.GalleryID == 0 {
			if galleryID, ok := GalleryIDFromWeb(comicInfo.Web); ok {
				archive.GalleryID = galleryID
			}
		}
		if archive.GalleryID == 0 {
			return nil
		}

		index.Archives = append(index.Archives, archive)
		return nil
	})
	if err != nil {
		return LibraryIndex{}, err
	}

	sort.Slice(index.Archives, func(i, j int) bool {
		return index.Archives[i].Path < index.Archives[j].Path
	})
	return index, nil
}

func (i LibraryIndex) ExistingGalleryPaths() map[int64]string {
	paths := make(map[int64]string, len(i.Archives))
	for _, archive := range i.Archives {
		if archive.GalleryID <= 0 || archive.Path == "" {
			continue
		}
		if _, exists := paths[archive.GalleryID]; exists {
			continue
		}
		paths[archive.GalleryID] = archive.Path
	}
	return paths
}

func (i *LibraryIndex) RemoveExpired(now time.Time, retentionDays int) ([]string, error) {
	if i == nil {
		return nil, nil
	}
	if retentionDays <= 0 {
		return nil, errors.New("retentionDays must be positive")
	}

	businessDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := businessDay.AddDate(0, 0, -(retentionDays - 1)).Format(fetchedDateLayout)

	removed := make([]string, 0)
	filtered := i.Archives[:0]
	for _, archive := range i.Archives {
		if archive.Path == "" || archive.FetchedDate == "" || archive.FetchedDate >= cutoff {
			filtered = append(filtered, archive)
			continue
		}

		if err := os.Remove(archive.Path); err != nil && !os.IsNotExist(err) {
			return removed, err
		}
		removed = append(removed, archive.Path)
	}

	i.Archives = filtered
	return removed, nil
}

func GalleryIDFromWeb(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return 0, false
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for index := len(segments) - 1; index >= 0; index-- {
		if galleryID, ok := parseGalleryID(segments[index]); ok {
			return galleryID, true
		}
	}

	return 0, false
}

func readArchiveComicInfo(path string) (archiveComicInfo, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return archiveComicInfo{}, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "ComicInfo.xml" {
			continue
		}

		stream, err := file.Open()
		if err != nil {
			return archiveComicInfo{}, err
		}
		defer stream.Close()

		payload, err := io.ReadAll(stream)
		if err != nil {
			return archiveComicInfo{}, err
		}

		var info archiveComicInfo
		if err := xml.Unmarshal(payload, &info); err != nil {
			return archiveComicInfo{}, err
		}
		return info, nil
	}

	return archiveComicInfo{}, nil
}
