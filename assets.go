package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
)

var (
	assetVersions   = make(map[string]string)
	assetVersionsMu sync.Mutex
)

// assetURL returns the public URL for a file in static/ with a content-hash
// query string, so Cloudflare and browsers refetch it whenever it changes.
// Hashes are computed once per process; a deploy restarts the server.
func assetURL(name string) string {
	assetVersionsMu.Lock()
	defer assetVersionsMu.Unlock()

	url := "/static/" + name
	version, ok := assetVersions[name]
	if !ok {
		data, err := os.ReadFile(filepath.Join("static", name))
		if err != nil {
			return url
		}
		sum := sha256.Sum256(data)
		version = hex.EncodeToString(sum[:])[:10]
		assetVersions[name] = version
	}
	return url + "?v=" + version
}
