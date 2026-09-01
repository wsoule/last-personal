package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func renderHomePage(t *testing.T, data PageData) string {
	t.Helper()
	parseSiteTemplates(t)

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "index.html", data); err != nil {
		t.Fatalf("rendering index.html: %v", err)
	}
	return buf.String()
}

func homePageData(projects []Project) PageData {
	latest, abandoned := latestAndAbandoned(projects)
	return PageData{
		BasePageData:   BasePageData{CurrentPage: "home"},
		Name:           "Wyat",
		AbandonedCount: abandoned,
		LatestProject:  latest,
	}
}

func TestPrimaryURLPrefersLiveThenGitHub(t *testing.T) {
	cases := []struct {
		name string
		p    Project
		want string
	}{
		{"live and github", Project{LiveURL: "https://live", GitHubURL: "https://gh"}, "https://live"},
		{"github only", Project{GitHubURL: "https://gh"}, "https://gh"},
		{"live only", Project{LiveURL: "https://live"}, "https://live"},
		{"neither", Project{}, ""},
	}
	for _, tc := range cases {
		if got := tc.p.PrimaryURL(); got != tc.want {
			t.Errorf("%s: PrimaryURL() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestLatestAndAbandonedCountsEverythingButTheNewest(t *testing.T) {
	latest, abandoned := latestAndAbandoned([]Project{
		{Name: "New"}, {Name: "Old"}, {Name: "Older"},
	})
	if latest.Name != "New" {
		t.Errorf("latest = %q, want New", latest.Name)
	}
	if abandoned != 2 {
		t.Errorf("abandoned = %d, want 2", abandoned)
	}

	latest, abandoned = latestAndAbandoned(nil)
	if latest.Name != "" || abandoned != 0 {
		t.Errorf("empty list: latest = %q, abandoned = %d; want zero values", latest.Name, abandoned)
	}
}

func TestHomePageHeadlineLinksCountAndLatestProject(t *testing.T) {
	// Use the real showcase list so the headline is checked against what
	// actually ships.
	projects := showcaseProjects()
	page := renderHomePage(t, homePageData(projects))

	wantCount := fmt.Sprintf(`<a href="/projects">%d</a>`, len(projects)-1)
	if !strings.Contains(page, wantCount) {
		t.Errorf("headline missing abandoned-count link %s", wantCount)
	}

	latest := projects[0]
	wantLatest := fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener">%s</a>`, latest.PrimaryURL(), latest.Name)
	if !strings.Contains(page, wantLatest) {
		t.Errorf("headline missing latest-project link %s", wantLatest)
	}

	if !strings.Contains(page, "I have abandoned") || !strings.Contains(page, "how long until I abandon") {
		t.Error("headline copy missing from rendered page")
	}
}

func TestHomePageHeadlineFallsBackToGitHubWhenNoLiveSite(t *testing.T) {
	page := renderHomePage(t, homePageData([]Project{
		{Name: "Query", Slug: "query", GitHubURL: "https://github.com/brass-raven/Query"},
		{Name: "Older", Slug: "older"},
	}))

	if !strings.Contains(page, `<a href="https://github.com/brass-raven/Query" target="_blank" rel="noopener">Query</a>`) {
		t.Error("headline should link the latest project to GitHub when it has no live site")
	}
}

func TestHomePageHeadlineFallsBackToProjectsAnchorWhenNoURLs(t *testing.T) {
	page := renderHomePage(t, homePageData([]Project{
		{Name: "Secret", Slug: "secret"},
	}))

	if !strings.Contains(page, `<a href="/projects#secret">Secret</a>`) {
		t.Error("headline should link to the project's section on /projects when it has no live or GitHub URL")
	}
	if !strings.Contains(page, `<a href="/projects">0</a>`) {
		t.Error("a single project means zero abandoned")
	}
}

func TestHomePageOmitsHeadlineWithoutProjects(t *testing.T) {
	page := renderHomePage(t, homePageData(nil))

	if strings.Contains(page, "I have abandoned") {
		t.Error("headline rendered with no projects to talk about")
	}
}
