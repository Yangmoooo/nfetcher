package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func FinalGalleryPath(root, fileName string) string {
	return filepath.Join(root, fileName)
}

func TempArchivePath(finalPath string, galleryID int64) string {
	return filepath.Join(filepath.Dir(finalPath), fmt.Sprintf(".%d.cbz.part", galleryID))
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func AtomicReplace(tempPath, finalPath string) error {
	if err := EnsureDir(filepath.Dir(finalPath)); err != nil {
		return err
	}
	return os.Rename(tempPath, finalPath)
}

func ExistingGalleryPaths(root string) (map[int64]string, error) {
	index, err := ScanLibraryIndex(root)
	if err != nil {
		return nil, err
	}
	return index.ExistingGalleryPaths(), nil
}

func GalleryIDFromPath(path string) (int64, bool) {
	if filepath.Ext(path) != ".cbz" {
		return 0, false
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if id, ok := parseGalleryID(base); ok {
		return id, true
	}

	index := strings.LastIndex(base, " - ")
	if index == -1 {
		return 0, false
	}

	return parseGalleryID(base[index+3:])
}

func parseGalleryID(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}
