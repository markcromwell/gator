package feed_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markcromwell/gator/internal/feed"
)

func TestFetchFeed_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server error")
	}))
	defer srv.Close()

	if _, err := feed.FetchFeed(context.Background(), srv.URL); err == nil {
		t.Fatalf("expected error for non-200 status")
	}
}

func TestFetchFeed_BadXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "<rss><channel><title>no close")
	}))
	defer srv.Close()

	if _, err := feed.FetchFeed(context.Background(), srv.URL); err == nil {
		t.Fatalf("expected xml parse error for malformed body")
	}
}
