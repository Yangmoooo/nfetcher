package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RunMode            string
	ScheduleCron       string
	Timezone           string
	LibraryDir         string
	RetentionDays      int
	SearchQuery        string
	SearchSort         string
	SearchPage         int
	GalleryConcurrency int
	DetailConcurrency  int
	PageConcurrency    int
	RequestRPS         float64
	RequestBurst       int
	HTTPTimeout        time.Duration
	RetryMax           int
}

func Load() (Config, error) {
	cfg := Config{
		RunMode:            getenv("RUN_MODE", "daemon"),
		ScheduleCron:       getenv("SCHEDULE_CRON", "0 18 * * *"),
		Timezone:           getenv("TZ", "Asia/Shanghai"),
		LibraryDir:         getenv("LIBRARY_DIR", "/library/nhentai-popular"),
		RetentionDays:      getenvInt("RETENTION_DAYS", 7),
		SearchQuery:        getenv("SEARCH_QUERY", "language:chinese"),
		SearchSort:         getenv("SEARCH_SORT", "popular-today"),
		SearchPage:         getenvInt("SEARCH_PAGE", 1),
		GalleryConcurrency: getenvInt("GALLERY_CONCURRENCY", 3),
		DetailConcurrency:  getenvInt("DETAIL_CONCURRENCY", 5),
		PageConcurrency:    getenvInt("PAGE_CONCURRENCY", 4),
		RequestRPS:         getenvFloat("REQUEST_RPS", 4),
		RequestBurst:       getenvInt("REQUEST_BURST", 8),
		HTTPTimeout:        getenvDuration("HTTP_TIMEOUT", 30*time.Second),
		RetryMax:           getenvInt("RETRY_MAX", 3),
	}

	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return Config{}, fmt.Errorf("load timezone: %w", err)
	}

	switch {
	case cfg.SearchPage < 1:
		return Config{}, fmt.Errorf("SEARCH_PAGE must be >= 1")
	case cfg.RetentionDays < 1:
		return Config{}, fmt.Errorf("RETENTION_DAYS must be >= 1")
	case cfg.GalleryConcurrency < 1:
		return Config{}, fmt.Errorf("GALLERY_CONCURRENCY must be >= 1")
	case cfg.DetailConcurrency < 1:
		return Config{}, fmt.Errorf("DETAIL_CONCURRENCY must be >= 1")
	case cfg.PageConcurrency < 1:
		return Config{}, fmt.Errorf("PAGE_CONCURRENCY must be >= 1")
	case cfg.RequestRPS <= 0:
		return Config{}, fmt.Errorf("REQUEST_RPS must be > 0")
	case cfg.RequestBurst < 1:
		return Config{}, fmt.Errorf("REQUEST_BURST must be >= 1")
	case cfg.HTTPTimeout <= 0:
		return Config{}, fmt.Errorf("HTTP_TIMEOUT must be > 0")
	case cfg.RetryMax < 0:
		return Config{}, fmt.Errorf("RETRY_MAX must be >= 0")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getenvFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
