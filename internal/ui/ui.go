package ui

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/CatOnAcidd/logship/internal/config"
	"github.com/CatOnAcidd/logship/internal/ingest"
	"github.com/CatOnAcidd/logship/internal/store"
	"github.com/CatOnAcidd/logship/internal/transform"
)

//go:embed static/*
var staticFS embed.FS

func NewRouter(db *store.DB, cfg *config.Config, tr *transform.Engine) *chi.Mux {
	r := ingest.Router(db, cfg, tr)
	// static UI
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFS, "static/index.html")
	})
	return r
}
