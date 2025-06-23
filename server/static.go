package server

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func serveStatic(w http.ResponseWriter, r *http.Request) error {
	path := fmt.Sprintf("static%s", r.URL.Path)
	if _, err := staticFS.ReadFile(path); err != nil {
		return err
	}
	http.ServeFileFS(w, r, staticFS, fmt.Sprintf("static%s", r.URL.Path))
	return nil
}
