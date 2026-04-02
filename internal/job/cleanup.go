package job

import (
	"os"
	"path/filepath"
	"time"
)

func CleanupOldDirs(root string, now time.Time, retentionDays int) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	businessDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := businessDay.AddDate(0, 0, -(retentionDays - 1))

	removed := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".tmp" {
			continue
		}

		day, err := time.ParseInLocation("2006-01-02", entry.Name(), now.Location())
		if err != nil {
			continue
		}

		if !day.Before(cutoff) {
			continue
		}

		doomedPath := filepath.Join(root, entry.Name())
		if err := os.RemoveAll(doomedPath); err != nil {
			return removed, err
		}

		removed = append(removed, doomedPath)
	}

	return removed, nil
}
