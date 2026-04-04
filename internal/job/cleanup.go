package job

import (
	"time"

	"nfetcher/internal/storage"
)

func CleanupExpired(index *storage.LibraryIndex, now time.Time, retentionDays int) ([]string, error) {
	return index.RemoveExpired(now, retentionDays)
}
