package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

// Embed everything under internal/ui/static
//go:embed static/*
var staticFS embed.FS

// Handler returns an http.Handler that serves the embedded UI.
// It serves index.html for "/" and lets FileServer handle other assets.
func Handler() http.Handler {
	sub, _ := fs.Sub(staticFS, "static")
	fsHandler := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html at root
		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, http.FS(sub), "index.html")
			return
		}
		// Otherwise serve static
		fsHandler.ServeHTTP(w, r)
	})
}
