package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"nfetcher/internal/config"
	"nfetcher/internal/httpx"
	"nfetcher/internal/summary"
)

type Bark struct {
	client    *httpx.Client
	baseURL   string
	deviceKey string
	sound     string
}

func NewBark(client *httpx.Client, cfg config.Config) *Bark {
	if client == nil || !cfg.BarkEnabled() {
		return nil
	}

	return &Bark{
		client:    client,
		baseURL:   strings.TrimRight(cfg.BarkBaseURL, "/"),
		deviceKey: cfg.BarkDeviceKey,
		sound:     cfg.BarkSound,
	}
}

func (b *Bark) Send(ctx context.Context, result summary.Result) error {
	if b == nil {
		return nil
	}

	endpoint, err := b.endpoint(result)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := b.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer httpx.DrainAndClose(resp.Body)
	return nil
}

func (b *Bark) endpoint(result summary.Result) (string, error) {
	title := url.PathEscape(formatBarkTitle(result))
	body := url.PathEscape(formatBarkBody(result))

	rawURL := fmt.Sprintf("%s/%s/%s/%s", b.baseURL, url.PathEscape(b.deviceKey), title, body)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	if b.sound != "" {
		query.Set("sound", b.sound)
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func formatBarkTitle(result summary.Result) string {
	return "Nfetcher"
}

func formatBarkBody(result summary.Result) string {
	lines := []string{
		fmt.Sprintf("Status: %s", result.Status()),
		fmt.Sprintf("Mode: %s", result.Mode),
		fmt.Sprintf("At: %s", result.StartedAtText()),
		"",
		fmt.Sprintf("Search | Dup | Queue: %d | %d | %d", result.SearchResults, result.Duplicates, result.Queued),
	}

	if result.Mode == "dry-run" {
		lines = append(
			lines,
			fmt.Sprintf("Warn | Fail: %d | %d", result.PreflightWarnings, result.PreflightFailures),
		)
	} else {
		lines = append(
			lines,
			fmt.Sprintf("Archived | Failed: %d | %d", result.ArchivedOK, result.ArchivedFailed),
		)
	}

	lines = append(lines, fmt.Sprintf("Duration: %s", result.DurationText()))

	if failed := result.FailedGalleryIDsText(8); failed != "-" {
		lines = append(lines, "Failed IDs: "+failed)
	}

	return strings.Join(lines, "\n")
}
