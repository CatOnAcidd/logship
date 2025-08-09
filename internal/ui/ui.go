package ui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/catonacidd/logship/internal/config"
)

//go:embed static/*
var staticFS embed.FS

// NewRouter builds a minimal router for Base and mounts static SPA UI.
func NewRouter(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	Mount(r)
	return r
}

// Mount serves the embedded static UI and an SPA-style index.html fallback.
func Mount(r *chi.Mux) {
	sub, _ := fs.Sub(staticFS, "static")
	fsrv := http.FileServer(http.FS(sub))

	// Static assets (CSS/JS/fonts/images)
	r.Handle("/static/*", http.StripPrefix("/static/", fsrv))

	// SPA fallback: for any other GET that's not an API or ingest path, return index.html
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// For API or ingest paths, let other handlers respond.
		if req.URL.Path == "/" || req.URL.Path == "" {
			// ok
		} else if len(req.URL.Path) >= 5 && req.URL.Path[:5] == "/api/" {
			http.NotFound(w, req)
			return
		} else if len(req.URL.Path) >= 7 && req.URL.Path[:7] == "/ingest" {
			http.NotFound(w, req)
			return
		}

		b, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, req, "index.html", time.Time{}, bytes.NewReader(b))
	})
}
