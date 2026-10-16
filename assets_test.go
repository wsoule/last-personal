package main

import (
	"regexp"
	"testing"
)

func TestAssetURLAddsContentHash(t *testing.T) {
	got := assetURL("style.css")
	if !regexp.MustCompile(`^/static/style\.css\?v=[0-9a-f]{10}$`).MatchString(got) {
		t.Fatalf("assetURL(style.css) = %q, want hashed query string", got)
	}
}

func TestAssetURLFallsBackForMissingFile(t *testing.T) {
	if got := assetURL("does-not-exist.css"); got != "/static/does-not-exist.css" {
		t.Fatalf("assetURL(missing) = %q, want plain path", got)
	}
}
