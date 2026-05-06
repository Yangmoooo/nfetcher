package config

import (
	"strings"
	"testing"
	"time"
)

func TestLibraryDirPathIsFixed(t *testing.T) {
	t.Setenv("LIBRARY_DIR", "/tmp/should-not-apply")

	if LibraryDirPath != "/nhentai-popular" {
		t.Fatalf("expected fixed library dir path, got %q", LibraryDirPath)
	}
}

func TestLoadRequiresNHentaiAPIKey(t *testing.T) {
	t.Setenv("TZ", "UTC")
	t.Setenv("NHENTAI_API_KEY", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NHENTAI_API_KEY is required") {
		t.Fatalf("expected API key error, got %v", err)
	}
}

func TestLoadReadsNHentaiAuthConfig(t *testing.T) {
	t.Setenv("TZ", "UTC")
	t.Setenv("NHENTAI_API_KEY", " test-key ")
	t.Setenv("NFETCHER_USER_AGENT", " nfetcher/test ")
	t.Setenv("DOWNLOAD_ISSUE_INTERVAL", "2m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.NHentaiAPIKey != "test-key" {
		t.Fatalf("expected trimmed API key, got %q", cfg.NHentaiAPIKey)
	}
	if cfg.UserAgent != "nfetcher/test" {
		t.Fatalf("expected trimmed user agent, got %q", cfg.UserAgent)
	}
	if cfg.DownloadIssueInterval != 2*time.Minute {
		t.Fatalf("expected download issue interval, got %s", cfg.DownloadIssueInterval)
	}
}
