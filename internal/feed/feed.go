package feed

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"
)

// RSSItem represents a single item in an RSS feed.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// RSSFeed is a minimal representation of an RSS document's channel and items.
type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

// FetchFeed fetches the RSS feed at feedURL, parses it into an RSSFeed struct,
// and returns the parsed result. The request uses a `User-Agent: gator` header
// and a reasonable timeout.
func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var parsed RSSFeed
	// Use a decoder and allow common HTML named entities that appear inside
	// some feeds (e.g. &ldquo;, &rdquo;). Leave standard XML entities alone.
	dec := xml.NewDecoder(bytes.NewReader(b))
	dec.Entity = map[string]string{
		"ldquo":  "\u201C",
		"rdquo":  "\u201D",
		"ndash":  "-",
		"mdash":  "-",
		"hellip": "\u2026",
	}
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("xml unmarshal: %w", err)
	}

	// Unescape HTML entities for the channel and each item
	parsed.Channel.Title = html.UnescapeString(parsed.Channel.Title)
	parsed.Channel.Description = html.UnescapeString(parsed.Channel.Description)
	for i := range parsed.Channel.Item {
		parsed.Channel.Item[i].Title = html.UnescapeString(parsed.Channel.Item[i].Title)
		parsed.Channel.Item[i].Description = html.UnescapeString(parsed.Channel.Item[i].Description)
	}

	return &parsed, nil
}
