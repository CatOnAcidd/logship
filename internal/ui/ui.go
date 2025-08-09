package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed static/*
var staticFS embed.FS

func Mount(r *chi.Mux) {
	sub, _ := fs.Sub(staticFS, "static")
	fsrv := http.FileServer(http.FS(sub))

	r.Handle("/static/*", http.StripPrefix("/static/", fsrv))

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/ingest") {
			http.NotFound(w, req); return
		}
		f, err := sub.Open("index.html")
		if err != nil { http.NotFound(w, req); return }
		defer f.Close()
		http.ServeContent(w, req, "index.html", /*modtime*/ 	/*ignored*/ 	/* zero */, f.(fs.File))
	})
}
