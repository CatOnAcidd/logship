package ui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed static/*
var content embed.FS

func Mount(r chi.Router) {
	sub, _ := fs.Sub(content, "static")
	r.Handle("/*", http.FileServerFS(sub))
}
