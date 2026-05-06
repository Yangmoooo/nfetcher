package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/time/rate"
)

type Shared struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	retryMax   int
}

type Client struct {
	shared  *Shared
	headers http.Header
}

type StatusError struct {
	URL        string
	StatusCode int
}

func (e StatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %d for %s", e.StatusCode, e.URL)
}

func APIHeaders(userAgent, apiKey string) http.Header {
	return http.Header{
		"User-Agent":    []string{userAgent},
		"Authorization": []string{"Key " + apiKey},
		"Accept":        []string{"application/json, text/plain, */*"},
	}
}

func DownloadHeaders(userAgent string) http.Header {
	return http.Header{
		"User-Agent": []string{userAgent},
		"Accept":     []string{"application/octet-stream, application/zip, */*"},
	}
}

func DefaultHeaders(userAgent string) http.Header {
	return http.Header{
		"User-Agent": []string{userAgent},
		"Accept":     []string{"application/json, text/plain, */*"},
	}
}

func NewShared(timeout time.Duration, rps float64, burst, retryMax int) *Shared {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment

	return &Shared{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		limiter:  rate.NewLimiter(rate.Limit(rps), burst),
		retryMax: retryMax,
	}
}

func NewClient(shared *Shared, headers http.Header) *Client {
	return &Client{
		shared:  shared,
		headers: headers.Clone(),
	}
}

func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.shared.retryMax; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, retryDelay(attempt-1)); err != nil {
				return nil, err
			}
		}

		cloned, err := cloneRequest(ctx, req)
		if err != nil {
			return nil, err
		}

		applyHeaders(cloned, c.headers)

		if err := c.shared.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := c.shared.httpClient.Do(cloned)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		err = StatusError{URL: cloned.URL.String(), StatusCode: resp.StatusCode}
		if retryableStatus(resp.StatusCode) && attempt < c.shared.retryMax {
			DrainAndClose(resp.Body)
			lastErr = err
			continue
		}

		DrainAndClose(resp.Body)
		return nil, err
	}

	if lastErr == nil {
		lastErr = errors.New("request failed without error details")
	}

	return nil, lastErr
}

func (c *Client) DownloadToFile(ctx context.Context, rawURL, dstPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer DrainAndClose(resp.Body)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	tempPath := dstPath + ".part"
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		_ = os.Remove(tempPath)
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return os.Rename(tempPath, dstPath)
}

func DrainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

func applyHeaders(req *http.Request, headers http.Header) {
	for key, values := range headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

func cloneRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
	cloned := req.Clone(ctx)
	if req.Body == nil {
		return cloned, nil
	}

	if req.GetBody == nil {
		return nil, errors.New("request body is not replayable")
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}

	cloned.Body = body
	return cloned, nil
}

func retryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryDelay(attempt int) time.Duration {
	delay := time.Second
	for index := 0; index < attempt; index++ {
		delay *= 3
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
