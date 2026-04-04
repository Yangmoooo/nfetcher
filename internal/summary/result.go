package summary

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusPartial Status = "partial"
	StatusFail    Status = "fail"
)

type Result struct {
	Mode              string
	Date              string
	Query             string
	Sort              string
	Page              int
	SearchResults     int
	Duplicates        int
	Queued            int
	ArchivedOK        int
	ArchivedFailed    int
	DetailErrors      int
	RemovedDirs       int
	PreflightWarnings int
	PreflightFailures int
	Duration          time.Duration
	FailedGalleryIDs  []int64
	ErrorCount        int
}

func (r Result) Status() Status {
	if r.ErrorCount == 0 && r.PreflightFailures == 0 {
		return StatusOK
	}

	if r.ArchivedOK > 0 || r.Duplicates > 0 || r.RemovedDirs > 0 {
		return StatusPartial
	}

	if r.Mode == "dry-run" && (r.SearchResults > 0 || r.Queued > 0 || r.DetailErrors > 0) {
		return StatusPartial
	}

	return StatusFail
}

func (r Result) DurationText() string {
	switch {
	case r.Duration <= 0:
		return "0s"
	case r.Duration < time.Second:
		return r.Duration.Round(time.Millisecond).String()
	default:
		return r.Duration.Round(time.Second).String()
	}
}

func (r Result) FailedGalleryIDsText(limit int) string {
	ids := uniqueSortedIDs(r.FailedGalleryIDs)
	if len(ids) == 0 {
		return "-"
	}

	if limit > 0 && len(ids) > limit {
		parts := make([]string, 0, limit+1)
		for _, id := range ids[:limit] {
			parts = append(parts, strconv.FormatInt(id, 10))
		}
		parts = append(parts, fmt.Sprintf("+%d more", len(ids)-limit))
		return strings.Join(parts, ",")
	}

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}

func (r Result) LogLine() string {
	return fmt.Sprintf(
		"summary status=%s mode=%s date=%s query=%q sort=%q page=%d search_results=%d duplicates=%d queued=%d archived_ok=%d archived_failed=%d detail_errors=%d removed_dirs=%d preflight_warnings=%d preflight_failures=%d error_count=%d duration=%s failed_gallery_ids=%q",
		r.Status(),
		r.Mode,
		r.Date,
		r.Query,
		r.Sort,
		r.Page,
		r.SearchResults,
		r.Duplicates,
		r.Queued,
		r.ArchivedOK,
		r.ArchivedFailed,
		r.DetailErrors,
		r.RemovedDirs,
		r.PreflightWarnings,
		r.PreflightFailures,
		r.ErrorCount,
		r.DurationText(),
		r.FailedGalleryIDsText(12),
	)
}

func uniqueSortedIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}

	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		set[id] = struct{}{}
	}

	unique := make([]int64, 0, len(set))
	for id := range set {
		unique = append(unique, id)
	}
	sort.Slice(unique, func(i, j int) bool {
		return unique[i] < unique[j]
	})
	return unique
}
