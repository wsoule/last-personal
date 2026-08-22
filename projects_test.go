package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var (
	anchorPattern = regexp.MustCompile(`href="#([^"]+)"`)
	idPattern     = regexp.MustCompile(`id="([^"]+)"`)
)

func renderProjectsPage(t *testing.T, projects []Project) string {
	t.Helper()
	parseSiteTemplates(t)

	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "projects.html", ProjectsPageData{
		BasePageData: BasePageData{CurrentPage: "projects"},
		Projects:     projects,
	})
	if err != nil {
		t.Fatalf("rendering projects.html: %v", err)
	}
	return buf.String()
}

func TestProjectsPageGivesEachProjectAnAnchorID(t *testing.T) {
	page := renderProjectsPage(t, []Project{
		{Name: "Loggit", Slug: "loggit", Description: "media tracker"},
		{Name: "Dispatch", Slug: "dispatch", Description: "agent control"},
	})

	for _, slug := range []string{"loggit", "dispatch"} {
		if !strings.Contains(page, fmt.Sprintf(`id="%s"`, slug)) {
			t.Errorf("rendered page missing id=%q", slug)
		}
	}
}

func TestProjectsPageHasNoDeadAnchors(t *testing.T) {
	// Render with the real project list so the page-toc is checked against
	// the sections it actually links to.
	page := renderProjectsPage(t, showcaseProjects())

	ids := map[string]bool{}
	for _, m := range idPattern.FindAllStringSubmatch(page, -1) {
		ids[m[1]] = true
	}

	for _, m := range anchorPattern.FindAllStringSubmatch(page, -1) {
		if target := m[1]; !ids[target] {
			t.Errorf("page-toc links to #%s but no element has that id", target)
		}
	}
}

func TestShowcaseProjectsIncludeLoggitAndDispatch(t *testing.T) {
	byName := map[string]Project{}
	for _, p := range showcaseProjects() {
		byName[p.Name] = p
	}

	loggit, ok := byName["Loggit"]
	if !ok {
		t.Fatal("showcaseProjects() missing Loggit")
	}
	if loggit.LiveURL != "https://useloggit.app" {
		t.Errorf("Loggit.LiveURL = %q, want https://useloggit.app", loggit.LiveURL)
	}
	// useloggit.app sends frame-ancestors 'self', so a preview would render blank.
	if loggit.IframeURL != "" {
		t.Errorf("Loggit.IframeURL = %q, want empty (the site refuses framing)", loggit.IframeURL)
	}

	dispatch, ok := byName["Dispatch"]
	if !ok {
		t.Fatal("showcaseProjects() missing Dispatch")
	}
	if dispatch.LiveURL != "https://dispatch.foo" {
		t.Errorf("Dispatch.LiveURL = %q, want https://dispatch.foo", dispatch.LiveURL)
	}
}

func TestEveryShowcaseProjectHasASlug(t *testing.T) {
	for _, p := range showcaseProjects() {
		if p.Slug == "" {
			t.Errorf("project %q has no Slug", p.Name)
		}
	}
}

func TestShowcaseProjectsNoLongerIncludeHydrogen(t *testing.T) {
	for _, p := range showcaseProjects() {
		if strings.EqualFold(p.Name, "Hydrogen") {
			t.Error("showcaseProjects() still includes Hydrogen")
		}
	}
}

func TestLoggitLinksToTheAppStore(t *testing.T) {
	var loggit Project
	for _, p := range showcaseProjects() {
		if p.Name == "Loggit" {
			loggit = p
		}
	}

	const want = "https://apps.apple.com/us/app/loggit-movies-books-games/id6766703437"
	if loggit.AppStoreURL != want {
		t.Errorf("Loggit.AppStoreURL = %q, want %q", loggit.AppStoreURL, want)
	}
}

func TestProjectsPageRendersAppStoreLink(t *testing.T) {
	page := renderProjectsPage(t, []Project{{
		Name:        "Loggit",
		Slug:        "loggit",
		AppStoreURL: "https://apps.apple.com/us/app/x/id1",
	}})

	if !strings.Contains(page, `href="https://apps.apple.com/us/app/x/id1"`) {
		t.Error("rendered page missing the App Store link")
	}
	if !strings.Contains(page, "App Store") {
		t.Error("rendered page missing the App Store link label")
	}
}

func TestProjectsPageOmitsAppStoreLinkWhenAbsent(t *testing.T) {
	page := renderProjectsPage(t, []Project{{Name: "ngmi", Slug: "ngmi", LiveURL: "https://ngmi.review"}})

	if strings.Contains(page, "App Store") {
		t.Error("rendered page shows an App Store link for a project that has none")
	}
}
