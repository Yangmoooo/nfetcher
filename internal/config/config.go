package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	LibraryDir            string
	RunMode               string
	ScheduleCron          string
	Timezone              string
	RetentionDays         int
	SearchQuery           string
	SearchSort            string
	SearchPage            int
	GalleryConcurrency    int
	RequestRPS            float64
	RequestBurst          int
	DownloadIssueInterval time.Duration
	HTTPTimeout           time.Duration
	RetryMax              int
	NHentaiAPIKey         string
	UserAgent             string
	BarkBaseURL           string
	BarkDeviceKey         string
	BarkSound             string
}

const (
	LibraryDirPath   = "/nhentai"
	defaultUserAgent = "nfetcher/1.0 (https://github.com/Yangmoooo/nfetcher)"
)

func Load() (Config, error) {
	cfg := Config{
		LibraryDir:            strings.TrimSpace(getenv("NF_LIBRARY_DIR", LibraryDirPath)),
		RunMode:               getenv("RUN_MODE", "daemon"),
		ScheduleCron:          getenv("SCHEDULE_CRON", "30 17 * * *"),
		Timezone:              getenv("TZ", "Asia/Shanghai"),
		RetentionDays:         getenvInt("RETENTION_DAYS", 7),
		SearchQuery:           getenv("SEARCH_QUERY", "language:chinese"),
		SearchSort:            getenv("SEARCH_SORT", "popular-today"),
		SearchPage:            getenvInt("SEARCH_PAGE", 1),
		GalleryConcurrency:    getenvInt("GALLERY_CONCURRENCY", 3),
		RequestRPS:            getenvFloat("REQUEST_RPS", 4),
		RequestBurst:          getenvInt("REQUEST_BURST", 8),
		DownloadIssueInterval: getenvDuration("DOWNLOAD_ISSUE_INTERVAL", 30*time.Second),
		HTTPTimeout:           getenvDuration("HTTP_TIMEOUT", 30*time.Second),
		RetryMax:              getenvInt("RETRY_MAX", 3),
		NHentaiAPIKey:         strings.TrimSpace(getenv("NF_NHENTAI_API_KEY", "")),
		UserAgent:             strings.TrimSpace(getenv("NFETCHER_USER_AGENT", defaultUserAgent)),
		BarkBaseURL:           strings.TrimSpace(getenv("NF_BARK_BASE_URL", "")),
		BarkDeviceKey:         strings.TrimSpace(getenv("NF_BARK_DEVICE_KEY", "")),
		BarkSound:             strings.TrimSpace(getenv("NF_BARK_SOUND", "healthnotification")),
	}

	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return Config{}, fmt.Errorf("load timezone: %w", err)
	}

	switch {
	case cfg.LibraryDir == "":
		return Config{}, fmt.Errorf("NF_LIBRARY_DIR must not be empty")
	case cfg.SearchPage < 1:
		return Config{}, fmt.Errorf("SEARCH_PAGE must be >= 1")
	case cfg.RetentionDays < 1:
		return Config{}, fmt.Errorf("RETENTION_DAYS must be >= 1")
	case cfg.GalleryConcurrency < 1:
		return Config{}, fmt.Errorf("GALLERY_CONCURRENCY must be >= 1")
	case cfg.RequestRPS <= 0:
		return Config{}, fmt.Errorf("REQUEST_RPS must be > 0")
	case cfg.RequestBurst < 1:
		return Config{}, fmt.Errorf("REQUEST_BURST must be >= 1")
	case cfg.DownloadIssueInterval < 0:
		return Config{}, fmt.Errorf("DOWNLOAD_ISSUE_INTERVAL must be >= 0")
	case cfg.HTTPTimeout <= 0:
		return Config{}, fmt.Errorf("HTTP_TIMEOUT must be > 0")
	case cfg.RetryMax < 0:
		return Config{}, fmt.Errorf("RETRY_MAX must be >= 0")
	case cfg.NHentaiAPIKey == "":
		return Config{}, fmt.Errorf("NF_NHENTAI_API_KEY is required")
	case cfg.UserAgent == "":
		return Config{}, fmt.Errorf("NFETCHER_USER_AGENT must not be empty")
	}

	if cfg.BarkBaseURL == "" && cfg.BarkDeviceKey != "" {
		return Config{}, fmt.Errorf("NF_BARK_BASE_URL is required when NF_BARK_DEVICE_KEY is set")
	}
	if cfg.BarkBaseURL != "" && cfg.BarkDeviceKey == "" {
		return Config{}, fmt.Errorf("NF_BARK_DEVICE_KEY is required when NF_BARK_BASE_URL is set")
	}
	if cfg.BarkBaseURL != "" {
		parsed, err := url.Parse(cfg.BarkBaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return Config{}, fmt.Errorf("NF_BARK_BASE_URL must be a valid absolute URL")
		}
	}

	return cfg, nil
}

func (c Config) BarkEnabled() bool {
	return c.BarkBaseURL != "" && c.BarkDeviceKey != ""
}

func (c Config) LibraryPath() string {
	if strings.TrimSpace(c.LibraryDir) == "" {
		return LibraryDirPath
	}
	return c.LibraryDir
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
