package config

import (
	"strings"
	"testing"
	"time"
)

func TestLibraryDirPathIsFixed(t *testing.T) {
	if LibraryDirPath != "/nhentai" {
		t.Fatalf("expected default library dir path, got %q", LibraryDirPath)
	}
}

func TestLoadReadsLibraryDir(t *testing.T) {
	t.Setenv("TZ", "UTC")
	t.Setenv("NF_NHENTAI_API_KEY", "test-key")
	t.Setenv("NF_LIBRARY_DIR", "./output")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.LibraryDir != "./output" {
		t.Fatalf("expected library dir ./output, got %q", cfg.LibraryDir)
	}
}

func TestLibraryPathFallsBackToContainerDefault(t *testing.T) {
	if got := (Config{}).LibraryPath(); got != LibraryDirPath {
		t.Fatalf("expected default library path %q, got %q", LibraryDirPath, got)
	}
}

func TestLoadRequiresNHentaiAPIKey(t *testing.T) {
	t.Setenv("TZ", "UTC")
	t.Setenv("NF_NHENTAI_API_KEY", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NF_NHENTAI_API_KEY is required") {
		t.Fatalf("expected API key error, got %v", err)
	}
}

func TestLoadReadsNHentaiAuthConfig(t *testing.T) {
	t.Setenv("TZ", "UTC")
	t.Setenv("NF_NHENTAI_API_KEY", " test-key ")
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
