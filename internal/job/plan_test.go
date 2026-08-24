package job

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"nfetcher/internal/config"
	"nfetcher/internal/nhentai"
)

type planRoundTripDoer struct {
	client  *http.Client
	baseURL *url.URL
}

func (d planRoundTripDoer) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = d.baseURL.Scheme
	cloned.URL.Host = d.baseURL.Host
	return d.client.Do(cloned)
}

func TestBuildPlanUsesSearchResultsWithoutFetchingDetails(t *testing.T) {
	detailRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/search" {
			detailRequested = true
			t.Fatalf("unexpected follow-up request %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result":[
				{"id":101,"media_id":"media-101","english_title":"Existing English","japanese_title":"Existing Japanese","num_pages":50},
				{"id":102,"media_id":"media-102","english_title":"Queued English","japanese_title":null,"num_pages":80},
				{"id":103,"media_id":"media-103","english_title":"Shorter English","japanese_title":"短いタイトル","num_pages":60}
			],
			"num_pages":1,
			"per_page":25,
			"total":3
		}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	client := nhentai.NewClient(planRoundTripDoer{client: server.Client(), baseURL: baseURL})
	runner := &Runner{
		Client: client,
		Config: config.Config{
			SearchQuery: "language:chinese",
			SearchSort:  "popular-today",
			SearchPage:  1,
		},
	}

	plan, err := runner.BuildPlan(context.Background(), PlanOptions{
		ExistingGalleryPaths: map[int64]string{101: "/library/existing.cbz"},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if detailRequested {
		t.Fatal("expected plan construction to use only the search response")
	}
	if plan.SearchResultsCount != 3 || len(plan.Duplicates) != 1 || len(plan.Queued) != 2 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if plan.Duplicates[0].Gallery.ID != 101 || plan.Duplicates[0].Rank != 1 {
		t.Fatalf("unexpected duplicate: %#v", plan.Duplicates[0])
	}
	if queued := plan.Queued[0]; queued.Gallery.ID != 102 || queued.Rank != 2 || queued.Gallery.Title.English != "Queued English" || queued.Gallery.NumPages != 80 {
		t.Fatalf("unexpected queued gallery: %#v", queued)
	}
	if queued := plan.Queued[1]; queued.Gallery.ID != 103 || queued.Rank != 3 || queued.Gallery.Title.Japanese != "短いタイトル" || queued.Gallery.NumPages != 60 {
		t.Fatalf("unexpected second queued gallery: %#v", queued)
	}
}
