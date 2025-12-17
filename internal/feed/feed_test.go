package feed_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markcromwell/gator/internal/feed"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8" ?>
<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
<channel>
  <title>RSS Feed Example &amp; &ldquo;Quote&rdquo;</title>
  <link>https://www.example.com</link>
  <description>This is an example RSS feed</description>
  <item>
    <title>First Article</title>
    <link>https://www.example.com/article1</link>
    <description>This is the content of the first article.</description>
    <pubDate>Mon, 06 Sep 2021 12:00:00 GMT</pubDate>
  </item>
  <item>
    <title>Second Article</title>
    <link>https://www.example.com/article2</link>
    <description>Here's the content of the second article.</description>
    <pubDate>Tue, 07 Sep 2021 14:30:00 GMT</pubDate>
  </item>
</channel>
</rss>`

func TestFetchFeed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintln(w, sampleRSS)
	}))
	defer srv.Close()

	f, err := feed.FetchFeed(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchFeed error: %v", err)
	}

	if f.Channel.Title == "" {
		t.Fatalf("expected channel title, got empty")
	}
	// Ensure HTML entities are unescaped (we included &ldquo; in title)
	if got := f.Channel.Title; got == "RSS Feed Example &ldquo;Quote&rdquo;" {
		t.Fatalf("expected unescaped title, got escaped: %s", got)
	}

	if len(f.Channel.Item) != 2 {
		t.Fatalf("expected 2 items, got %d", len(f.Channel.Item))
	}
}
