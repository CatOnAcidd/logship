package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/ingest"
	"github.com/catonacidd/logship/internal/store"
	"github.com/go-chi/chi/v5"
)

//go:embed web/*
var uiFS embed.FS

func Router(db *store.DB, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// API
	r.Get("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		s, err := db.Stats(r.Context())
		if err != nil {
			http.Error(w, "stats error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		_ = jsonNewEncoder(w).Encode(s)
	})
	r.Get("/api/events", ingest.HandleQuery(db))
	r.Post("/api/ingest", ingest.HandleHTTPIngest(db, cfg))

	// UI
	sub, _ := fs.Sub(uiFS, "web")
	fileServer := http.FileServer(http.FS(sub))
	// index
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, sub, "index.html")
	})
	// assets
	r.Handle("/*", fileServer)

	return r
}

// tiny local wrapper to avoid importing encoding/json here and in other files repeatedly
type _jsonEncoder interface{ Encode(v any) error }
func jsonNewEncoder(w http.ResponseWriter) _jsonEncoder { return &je{w: w} }

type je struct{ w http.ResponseWriter }

func (j *je) Encode(v any) error {
	j.w.Header().Set("content-type", "application/json")
	return jsonEnc(j.w, v)
}
