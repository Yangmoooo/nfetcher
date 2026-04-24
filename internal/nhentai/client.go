package nhentai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Doer interface {
	Do(context.Context, *http.Request) (*http.Response, error)
}

type Client struct {
	apiBase   string
	imageBase string
	apiClient Doer
}

func NewClient(apiClient Doer) *Client {
	return &Client{
		apiBase:   "https://nhentai.net",
		imageBase: "https://i.nhentai.net",
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

func (c *Client) GetGallery(ctx context.Context, id int64) (Gallery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v2/galleries/%d", c.apiBase, id), nil)
	if err != nil {
		return Gallery{}, err
	}

	resp, err := c.apiClient.Do(ctx, req)
	if err != nil {
		return Gallery{}, err
	}
	defer resp.Body.Close()

	var out Gallery
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Gallery{}, err
	}

	return out, nil
}

func (c *Client) ImageURL(pagePath string) string {
	return c.imageBase + "/" + strings.TrimLeft(pagePath, "/")
}
