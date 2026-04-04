package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func DateDir(root string, now time.Time) string {
	return filepath.Join(root, now.Format("2006-01-02"))
}

func FinalGalleryDir(root string, now time.Time, dirName string) string {
	return filepath.Join(DateDir(root, now), dirName)
}

func FinalGalleryPath(root string, now time.Time, dirName, fileName string) string {
	return filepath.Join(FinalGalleryDir(root, now, dirName), fileName)
}

func StageDir(now time.Time, galleryID int64) string {
	return filepath.Join(os.TempDir(), "nfetcher", now.Format("2006-01-02"), fmt.Sprintf("%d", galleryID))
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
	paths := make(map[int64]string)

	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return paths, nil
		}
		return nil, err
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if entry.Name() == ".tmp" {
				return filepath.SkipDir
			}
			return nil
		}

		galleryID, ok := GalleryIDFromPath(path)
		if !ok {
			return nil
		}

		if _, exists := paths[galleryID]; !exists {
			paths[galleryID] = path
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
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
