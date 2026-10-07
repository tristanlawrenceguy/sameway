package server

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/design"
)

// icon serves Sameway's icon from design/brand (tools/icons draws it), so
// its tab is told from the others; a browser asks for /favicon.ico on its
// own, and every page names /favicon.svg.
func icon(name, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := design.FS.ReadFile("brand/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", kind)
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(data)
	}
}

func iconRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /favicon.svg", icon("icon.svg", "image/svg+xml"))
	m.HandleFunc("GET /favicon.ico", icon("icon.ico", "image/x-icon"))
}
