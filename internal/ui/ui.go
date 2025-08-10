package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// Embed everything under internal/ui/static
//go:embed static/*
var staticFS embed.FS

// Handler returns an http.Handler that serves the embedded UI.
// It serves index.html at "/" and falls back to index.html for unknown routes (SPA-friendly).
func Handler() http.Handler {
	sub, _ := fs.Sub(staticFS, "static") // sub implements fs.FS
	fsHandler := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Root -> index.html
		if path == "" || path == "index.html" {
			http.ServeFileFS(w, r, sub, "index.html") // NOTE: pass fs.FS (sub), not http.FS(...)
			return
		}

		// Try to serve a static asset
		if f, err := sub.Open(path); err == nil {
			_ = f.Close()
			fsHandler.ServeHTTP(w, r)
			return
		}

		// SPA fallback -> index.html
		http.ServeFileFS(w, r, sub, "index.html")
	})
}
