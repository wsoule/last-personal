package main

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
<channel>
	<title><![CDATA[Wyat Does A Thing]]></title>
	<link>https://wyatdoesathing.substack.com</link>
	<description><![CDATA[Notes on building things]]></description>
	<item>
		<title><![CDATA[Shipping beats planning]]></title>
		<link>https://wyatdoesathing.substack.com/p/shipping-beats-planning</link>
		<description><![CDATA[<p>A short note on why I stopped writing specs.</p>]]></description>
		<pubDate>Tue, 12 Aug 2026 14:03:11 GMT</pubDate>
	</item>
	<item>
		<title><![CDATA[On reading the manual]]></title>
		<link>https://wyatdoesathing.substack.com/p/on-reading-the-manual</link>
		<description><![CDATA[<p>Turns out it was documented.</p>]]></description>
		<pubDate>Mon, 04 Aug 2026 09:00:00 +0000</pubDate>
	</item>
</channel>
</rss>`

func TestParseSubstackFeedExtractsPosts(t *testing.T) {
	posts, err := parseSubstackFeed([]byte(sampleFeed))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("len(posts) = %d, want 2", len(posts))
	}

	got := posts[0]
	if got.Title != "Shipping beats planning" {
		t.Errorf("Title = %q, want %q", got.Title, "Shipping beats planning")
	}
	if got.Link != "https://wyatdoesathing.substack.com/p/shipping-beats-planning" {
		t.Errorf("Link = %q, want the post permalink", got.Link)
	}
	want := time.Date(2026, time.August, 12, 14, 3, 11, 0, time.UTC)
	if !got.Published.Equal(want) {
		t.Errorf("Published = %v, want %v", got.Published, want)
	}
}

func TestParseSubstackFeedParsesNumericTimezoneOffset(t *testing.T) {
	posts, err := parseSubstackFeed([]byte(sampleFeed))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}

	// Substack emits both "GMT" and "+0000" forms; the second item uses "+0000".
	want := time.Date(2026, time.August, 4, 9, 0, 0, 0, time.UTC)
	if !posts[1].Published.Equal(want) {
		t.Errorf("Published = %v, want %v", posts[1].Published, want)
	}
}

func TestParseSubstackFeedStripsHTMLFromDescription(t *testing.T) {
	posts, err := parseSubstackFeed([]byte(sampleFeed))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}

	want := "A short note on why I stopped writing specs."
	if posts[0].Description != want {
		t.Errorf("Description = %q, want %q", posts[0].Description, want)
	}
}

func TestParseSubstackFeedSortsNewestFirst(t *testing.T) {
	// Same feed with the items in reverse chronological disorder.
	outOfOrder := `<?xml version="1.0"?><rss version="2.0"><channel>
		<item><title>older</title><link>a</link><pubDate>Mon, 04 Aug 2026 09:00:00 GMT</pubDate></item>
		<item><title>newer</title><link>b</link><pubDate>Tue, 12 Aug 2026 09:00:00 GMT</pubDate></item>
	</channel></rss>`

	posts, err := parseSubstackFeed([]byte(outOfOrder))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}
	if posts[0].Title != "newer" {
		t.Errorf("posts[0].Title = %q, want %q", posts[0].Title, "newer")
	}
}

func TestParseSubstackFeedEmptyChannelIsNotAnError(t *testing.T) {
	empty := `<?xml version="1.0"?><rss version="2.0"><channel>
		<title>Wyat Does A Thing</title></channel></rss>`

	posts, err := parseSubstackFeed([]byte(empty))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("len(posts) = %d, want 0", len(posts))
	}
}

func TestParseSubstackFeedRejectsNonFeedBody(t *testing.T) {
	// A publication with no posts redirects to an HTML profile page.
	_, err := parseSubstackFeed([]byte("<!DOCTYPE html><html><body>profile</body></html>"))
	if err == nil {
		t.Fatal("parseSubstackFeed(html) returned nil error, want an error")
	}
}

// --- cache -------------------------------------------------------------

func TestSubstackCacheServesFromCacheWithinTTL(t *testing.T) {
	calls := 0
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	c := &substackCache{
		ttl: 30 * time.Minute,
		now: func() time.Time { return now },
		fetch: func() ([]SubstackPost, error) {
			calls++
			return []SubstackPost{{Title: "one"}}, nil
		},
	}

	c.Posts()
	now = now.Add(29 * time.Minute)
	posts := c.Posts()

	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
	if len(posts) != 1 || posts[0].Title != "one" {
		t.Errorf("Posts() = %+v, want the cached post", posts)
	}
}

func TestSubstackCacheRefetchesAfterTTL(t *testing.T) {
	calls := 0
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	c := &substackCache{
		ttl: 30 * time.Minute,
		now: func() time.Time { return now },
		fetch: func() ([]SubstackPost, error) {
			calls++
			return []SubstackPost{{Title: "fresh"}}, nil
		},
	}

	c.Posts()
	now = now.Add(31 * time.Minute)
	c.Posts()

	if calls != 2 {
		t.Errorf("fetch called %d times, want 2", calls)
	}
}

func TestSubstackCacheServesStalePostsWhenFetchFails(t *testing.T) {
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	fail := false
	c := &substackCache{
		ttl: 30 * time.Minute,
		now: func() time.Time { return now },
		fetch: func() ([]SubstackPost, error) {
			if fail {
				return nil, errors.New("substack is down")
			}
			return []SubstackPost{{Title: "good"}}, nil
		},
	}

	c.Posts()
	fail = true
	now = now.Add(31 * time.Minute)
	posts := c.Posts()

	if len(posts) != 1 || posts[0].Title != "good" {
		t.Errorf("Posts() = %+v, want the last good post to survive a failed refresh", posts)
	}
}

func TestSubstackCacheReturnsNoPostsWhenFetchFailsWithNothingCached(t *testing.T) {
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	c := &substackCache{
		ttl: 30 * time.Minute,
		now: func() time.Time { return now },
		fetch: func() ([]SubstackPost, error) {
			return nil, errors.New("substack is down")
		},
	}

	if posts := c.Posts(); len(posts) != 0 {
		t.Fatalf("Posts() = %+v, want no posts", posts)
	}
}

func TestSubstackCacheDoesNotRetryFailedFetchUntilTTLExpires(t *testing.T) {
	calls := 0
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	c := &substackCache{
		ttl: 30 * time.Minute,
		now: func() time.Time { return now },
		fetch: func() ([]SubstackPost, error) {
			calls++
			return nil, errors.New("substack is down")
		},
	}

	c.Posts()
	now = now.Add(1 * time.Minute)
	c.Posts()

	// A failing feed must not mean a blocking HTTP call on every page load.
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
}

func TestParseSubstackFeedTruncatesLongDescription(t *testing.T) {
	// Substack often puts the whole post body in <description>.
	long := strings.Repeat("word ", 200)
	feed := `<?xml version="1.0"?><rss version="2.0"><channel><item>
		<title>long one</title><link>a</link>
		<description><![CDATA[<p>` + long + `</p>]]></description>
		<pubDate>Tue, 12 Aug 2026 09:00:00 GMT</pubDate>
	</item></channel></rss>`

	posts, err := parseSubstackFeed([]byte(feed))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}

	got := posts[0].Description
	if len(got) > 200 {
		t.Errorf("len(Description) = %d, want <= 200", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("Description = %q, want it to end with an ellipsis", got)
	}
	if strings.Contains(got, "wor…") {
		t.Errorf("Description = %q, want truncation on a word boundary", got)
	}
}

func TestParseSubstackFeedKeepsShortDescriptionIntact(t *testing.T) {
	posts, err := parseSubstackFeed([]byte(sampleFeed))
	if err != nil {
		t.Fatalf("parseSubstackFeed returned error: %v", err)
	}

	if strings.HasSuffix(posts[0].Description, "…") {
		t.Errorf("Description = %q, want no ellipsis on a short description", posts[0].Description)
	}
}

// --- handler wiring ----------------------------------------------------

// parseSiteTemplates loads the real templates the way main() does.
func parseSiteTemplates(t *testing.T) {
	t.Helper()
	funcMap := template.FuncMap{"currentYear": func() int { return 2026 }}
	templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	template.Must(templates.ParseGlob("templates/partials/*.html"))
	template.Must(templates.ParseGlob("templates/finance/*.html"))
}

func stubSubstack(posts []SubstackPost, err error) {
	substackFeed = &substackCache{
		ttl:   time.Hour,
		now:   time.Now,
		fetch: func() ([]SubstackPost, error) { return posts, err },
	}
}

func TestBlogsHandlerRendersSubstackPosts(t *testing.T) {
	parseSiteTemplates(t)
	stubSubstack([]SubstackPost{{
		Title:       "Shipping beats planning",
		Link:        "https://wyatdoesathing.substack.com/p/shipping",
		Published:   time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
		Description: "A short note on specs.",
	}}, nil)

	rec := httptest.NewRecorder()
	blogsHandler(rec, httptest.NewRequest(http.MethodGet, "/blogs", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"From Substack",
		"Shipping beats planning",
		"https://wyatdoesathing.substack.com/p/shipping",
		"A short note on specs.",
		"Aug 12, 2026",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestBlogsHandlerStillRendersLocalPostsWhenSubstackIsDown(t *testing.T) {
	parseSiteTemplates(t)
	stubSubstack(nil, errors.New("substack is down"))

	rec := httptest.NewRecorder()
	blogsHandler(rec, httptest.NewRequest(http.MethodGet, "/blogs", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Posts") {
		t.Error("body missing the local Posts section")
	}
	if strings.Contains(body, "From Substack") {
		t.Error("body should omit the Substack section entirely when there are no posts")
	}
}
