package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var content embed.FS

func Handler() (http.Handler, error) {
	sub, err := fs.Sub(content, "ui")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(sub)), nil
}
