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

// appManifest lets a browser install Sameway as an app of its own: a
// window without tabs or an address bar, its icon in the taskbar, the
// Start menu or the Dock, and on a phone's home screen (app_install.go).
func appManifest(w http.ResponseWriter, r *http.Request) { manifestStarting(w, "/", "") }

// manifestStarting is the manifest with the address the app opens at:
// for a phone on the Wi-Fi, one that pairs the app itself (phone_install.go),
// and any shortcuts that phone has beyond everyone's.
func manifestStarting(w http.ResponseWriter, start, more string) {
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(`{"name":"Sameway","short_name":"Sameway","start_url":"` + start + `","scope":"/","display":"standalone",` +
		`"background_color":"#ffffff","theme_color":"#1a45a8","description":"Your workspace, with an assistant",` +
		// Held, the app's icon offers what a phone opens it for most.
		`"id":"/","shortcuts":[{"name":"Save to Sameway","short_name":"Save","description":"Add photos, files, a link or a few words","url":"/share"},` +
		`{"name":"Chat with the assistant","short_name":"Chat","url":"/chat#message"},{"name":"Today","short_name":"Today","url":"/today"},{"name":"Search","short_name":"Search","url":"/search"}` + more + `],` +
		`"share_target":{"action":"/share","method":"POST","enctype":"multipart/form-data","params":{"title":"title","text":"text","url":"url","files":[{"name":"files","accept":["image/*","application/pdf","text/*","audio/*","video/*"]}]}},` +
		`"icons":[{"src":"/icon-192.png","sizes":"192x192","type":"image/png"},{"src":"/icon-512.png","sizes":"512x512","type":"image/png"},` +
		`{"src":"/icon-square-512.png","sizes":"512x512","type":"image/png","purpose":"maskable"}]}`))
}
