package nhentai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Doer interface {
	Do(context.Context, *http.Request) (*http.Response, error)
}

type Client struct {
	apiBase   string
	apiClient Doer
}

func NewClient(apiClient Doer) *Client {
	return &Client{
		apiBase:   "https://nhentai.net",
		apiClient: apiClient,
	}
}

func (c *Client) Search(ctx context.Context, query, sort string, page int) (SearchResponse, error) {
	values := url.Values{}
	values.Set("query", query)
	values.Set("sort", sort)
	values.Set("page", fmt.Sprintf("%d", page))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBase+"/api/v2/search?"+values.Encode(), nil)
	if err != nil {
		return SearchResponse{}, err
	}

	resp, err := c.apiClient.Do(ctx, req)
	if err != nil {
		return SearchResponse{}, err
	}
	defer resp.Body.Close()

	var out SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return SearchResponse{}, err
	}
	out.Normalize()

	return out, nil
}

func (c *Client) DownloadGallery(ctx context.Context, id int64) (DownloadResponse, error) {
	values := url.Values{}
	values.Set("format", "cbz")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/v2/galleries/%d/download?%s", c.apiBase, id, values.Encode()), nil)
	if err != nil {
		return DownloadResponse{}, err
	}

	resp, err := c.apiClient.Do(ctx, req)
	if err != nil {
		return DownloadResponse{}, err
	}
	defer resp.Body.Close()

	var out DownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return DownloadResponse{}, err
	}
	if out.URL == "" {
		return DownloadResponse{}, fmt.Errorf("download url is empty for gallery %d", id)
	}

	return out, nil
}
