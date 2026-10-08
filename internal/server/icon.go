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
	m.HandleFunc("GET /icon-192.png", icon("icon-192.png", "image/png"))
	m.HandleFunc("GET /icon-512.png", icon("icon-512.png", "image/png"))
	m.HandleFunc("GET /icon-square-180.png", icon("icon-square-180.png", "image/png"))
	m.HandleFunc("GET /icon-square-512.png", icon("icon-square-512.png", "image/png"))
	m.HandleFunc("GET /manifest.webmanifest", appManifest)
}

// appManifest lets a browser install Sameway as an app of its own: a
// window without tabs or an address bar, its icon in the taskbar, the
// Start menu or the Dock, and on a phone's home screen (app_install.go).
func appManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Write([]byte(`{"name":"Sameway","short_name":"Sameway","start_url":"/","scope":"/","display":"standalone",` +
		`"background_color":"#ffffff","theme_color":"#1a45a8","description":"Your workspace, with an assistant",` +
		`"share_target":{"action":"/share","method":"POST","enctype":"multipart/form-data","params":{"title":"title","text":"text","url":"url","files":[{"name":"files","accept":["image/*","application/pdf","text/*","audio/*","video/*"]}]}},` +
		`"icons":[{"src":"/icon-192.png","sizes":"192x192","type":"image/png"},{"src":"/icon-512.png","sizes":"512x512","type":"image/png"},` +
		`{"src":"/icon-square-512.png","sizes":"512x512","type":"image/png","purpose":"maskable"}]}`))
}
