package main

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// SubstackPost is a single post pulled from the Substack RSS feed.
type SubstackPost struct {
	Title       string
	Link        string
	Published   time.Time
	Description string
}

// PublishedOn renders the publication date for display.
func (p SubstackPost) PublishedOn() string {
	if p.Published.IsZero() {
		return ""
	}
	return p.Published.Format("Jan 02, 2006")
}

// rssFeed mirrors the subset of RSS 2.0 that Substack emits.
type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

// pubDateLayouts covers the RFC 822 variants Substack uses ("GMT" and "+0000").
var pubDateLayouts = []string{time.RFC1123Z, time.RFC1123}

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// parseSubstackFeed turns RSS bytes into posts, newest first. It returns an
// error for anything that isn't an RSS document — a Substack publication with
// no posts serves an HTML profile page instead of a feed.
func parseSubstackFeed(data []byte) ([]SubstackPost, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing substack feed: %w", err)
	}

	posts := make([]SubstackPost, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		posts = append(posts, SubstackPost{
			Title:       strings.TrimSpace(item.Title),
			Link:        strings.TrimSpace(item.Link),
			Published:   parsePubDate(item.PubDate),
			Description: stripHTML(item.Description),
		})
	}

	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].Published.After(posts[j].Published)
	})

	return posts, nil
}

// parsePubDate returns the zero time for dates it cannot read, so one bad item
// never takes down the whole feed.
func parsePubDate(value string) time.Time {
	value = strings.TrimSpace(value)
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// descriptionLimit caps the excerpt length in bytes, ellipsis included.
// Substack frequently puts the entire post body in <description>.
const descriptionLimit = 200

const ellipsis = "…"

// stripHTML reduces a Substack description to a plain-text excerpt.
func stripHTML(s string) string {
	text := strings.TrimSpace(html.UnescapeString(htmlTagPattern.ReplaceAllString(s, "")))
	return truncateWords(text, descriptionLimit)
}

// truncateWords shortens s to at most limit bytes, cutting on a word boundary
// and appending an ellipsis. Short strings are returned untouched.
func truncateWords(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	cut := s[:limit-len(ellipsis)]
	if i := strings.LastIndexByte(cut, ' '); i > 0 {
		cut = cut[:i]
	}

	// Guard against slicing through a multi-byte rune when there was no space.
	cut = strings.ToValidUTF8(cut, "")

	return strings.TrimRight(cut, " ,;:.") + ellipsis
}

// substackCache serves posts from memory, refreshing at most once per ttl.
// A failed refresh keeps the last good posts and still resets the clock, so a
// broken feed never means a blocking HTTP call on every page load.
type substackCache struct {
	mu      sync.RWMutex
	posts   []SubstackPost
	fetched time.Time

	ttl   time.Duration
	now   func() time.Time
	fetch func() ([]SubstackPost, error)
}

// Posts returns the cached posts, refreshing first if the cache has expired.
func (c *substackCache) Posts() []SubstackPost {
	c.mu.RLock()
	fresh := !c.fetched.IsZero() && c.now().Sub(c.fetched) < c.ttl
	posts := c.posts
	c.mu.RUnlock()

	if fresh {
		return posts
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Another goroutine may have refreshed while we waited for the lock.
	if !c.fetched.IsZero() && c.now().Sub(c.fetched) < c.ttl {
		return c.posts
	}

	c.fetched = c.now()
	fetched, err := c.fetch()
	if err != nil {
		log.Println("Error refreshing Substack feed:", err)
		return c.posts
	}

	c.posts = fetched
	return c.posts
}

// newSubstackCache builds a cache backed by the live feed at feedURL.
func newSubstackCache(feedURL string, ttl time.Duration) *substackCache {
	return &substackCache{
		ttl:   ttl,
		now:   time.Now,
		fetch: func() ([]SubstackPost, error) { return fetchSubstackFeed(feedURL) },
	}
}

// fetchSubstackFeed retrieves and parses the feed at feedURL.
func fetchSubstackFeed(feedURL string) ([]SubstackPost, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("fetching substack feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("substack feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("reading substack feed: %w", err)
	}

	return parseSubstackFeed(body)
}
