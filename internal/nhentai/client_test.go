package nhentai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripDoer struct {
	client *http.Client
}

func (d roundTripDoer) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	return d.client.Do(req)
}

func TestSearchParsesCurrentGalleryListResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "language:chinese" {
			t.Fatalf("unexpected query %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result":[{
				"id":645649,
				"media_id":"3902888",
				"english_title":"English title",
				"japanese_title":"Japanese title",
				"num_pages":197,
				"tag_ids":[2937,29963],
				"blacklisted":false
			}],
			"num_pages":20,
			"per_page":25,
			"total":500
		}`))
	}))
	defer server.Close()

	client := NewClient(roundTripDoer{client: server.Client()})
	client.apiBase = server.URL

	response, err := client.Search(context.Background(), "language:chinese", "popular-today", 1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if response.Total != 500 || response.PerPage != 25 || response.NumPages != 20 {
		t.Fatalf("unexpected pagination: %#v", response)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one result, got %#v", response.Result)
	}

	gallery := response.Result[0]
	if gallery.ID != 645649 || gallery.MediaID != "3902888" || gallery.NumPages != 197 {
		t.Fatalf("unexpected gallery result: %#v", gallery)
	}
	if gallery.Title.Japanese != "Japanese title" || gallery.Title.English != "English title" {
		t.Fatalf("expected title fields mapped, got %#v", gallery.Title)
	}
}

func TestDownloadGalleryRequestsOfficialCBZURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/api/v2/galleries/645649/download" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "cbz" {
			t.Fatalf("unexpected format %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://download.example/645649.cbz","expires_at":1778046612}`))
	}))
	defer server.Close()

	client := NewClient(roundTripDoer{client: server.Client()})
	client.apiBase = server.URL

	response, err := client.DownloadGallery(context.Background(), 645649)
	if err != nil {
		t.Fatalf("download gallery: %v", err)
	}

	if response.URL != "https://download.example/645649.cbz" || response.ExpiresAt != 1778046612 {
		t.Fatalf("unexpected download response: %#v", response)
	}
}
