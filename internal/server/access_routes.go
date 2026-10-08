package server

import (
	"net/http"
	"strings"
)

// Who may use each route, said for every route here. It used to be a list
// of address prefixes that were the owner's, with everything else open to
// anyone let in, so a route whose address matched no prefix was shared by
// default: GET /api/workspaces, the other workspaces on this machine, was
// one until it was added to the list by hand. Now a route nobody said
// anything about is the owner's alone, and TestEveryRouteSaysWhoMayUseIt
// fails until it is said.

// routeFor says who may use a route.
type routeFor int

const (
	// people are everyone let in: those who may look read, and those who
	// may change it change, except the owner's own kinds of record.
	people routeFor = iota
	// owner is the workspace's owner alone.
	owner
)

var routeAccess = map[string]routeFor{
	"GET /api/changes":                 people,
	"GET /api/workspaces":              owner,
	"/":                                people,
	"GET /{$}":                         people,
	"GET /c/{canvas}":                  people,
	"GET /chat":                        people,
	"GET /help":                        people,
	"POST /help/set":                   owner,
	"GET /canvas/{id}":                 people,
	"POST /canvas/measure":             people, // and only those who may change it: measure.go
	"POST /chat":                       people,
	"POST /chat/stream":                people,
	"POST /chat/stop":                  people,
	"GET /chat/live":                   people,
	"POST /model/use":                  owner,
	"POST /t/{type}/add":               people,
	"POST /model/check":                owner,
	"GET /model/wait":                  owner,
	"POST /model/key":                  owner,
	"POST /model/ollama":               owner,
	"POST /chat/clear":                 people,
	"POST /chat/new":                   people,
	"POST /chat/open":                  people,
	"POST /chat/delete":                people,
	"POST /proposal/{id}/accept":       owner,
	"POST /proposal/{id}/dismiss":      owner,
	"POST /proposal/{id}/instead":      owner,
	"POST /activity/{id}/undo":         owner,
	"POST /suggestions/{id}/{answer}":  people,
	"POST /suggestions/accept-all":     people,
	"POST /act/{id}":                   people,
	"POST /canvas/{id}/props":          people,
	"POST /canvas/{id}/keep":           people,
	"POST /canvas/{id}/delete":         people,
	"GET /activity":                    owner,
	"POST /clock/set":                  people,
	"POST /habit/{id}/log":             people,
	"POST /clock/{id}/done":            people,
	"POST /clock/{id}/snooze":          people,
	"GET /clock/stream":                people,
	"POST /sync":                       people,
	"GET /events":                      people,
	"GET /workspaces":                  owner,
	"POST /workspaces/start":           owner,
	"GET /workspaces/new":              owner,
	"POST /workspaces/new":             owner,
	"GET /workspaces/copy":             owner,
	"POST /workspaces/copy":            owner,
	"GET /workspaces/delete":           owner,
	"POST /workspaces/delete":          owner,
	"POST /workspaces/restore":         owner,
	"GET /templates":                   people,
	"POST /templates/use":              people,
	"POST /quit":                       owner,
	"GET /bring":                       owner,
	"POST /bring":                      owner,
	"GET /calendars":                   owner,
	"POST /calendars/add":              owner,
	"POST /calendars/remove":           owner,
	"POST /phone/on":                   owner,
	"POST /phone/off":                  owner,
	"POST /phone/forget":               owner,
	"GET /today":                       people,
	"POST /brief":                      owner,
	"POST /workspaces/example":         owner,
	"POST /feedback":                   owner,
	"POST /backup/cloud":               owner,
	"POST /workspaces/from-copy":       owner,
	"POST /at-login":                   owner,
	"POST /restart":                    owner,
	"POST /notify/phone":               owner,
	"GET /search":                      people,
	"GET /when":                        people,
	"GET /design":                      people,
	"GET /design/sameway.css":          people,
	"GET /favicon.svg":                 people,
	"GET /favicon.ico":                 people,
	"GET /icon-192.png":                people,
	"GET /icon-512.png":                people,
	"GET /icon-square-180.png":         people,
	"GET /icon-square-512.png":         people,
	"GET /manifest.webmanifest":        people,
	"GET /design/sameway.js":           people,
	"GET /design/base/{file}":          people,
	"GET /t/{type}":                    people,
	"GET /t/{type}/import":             owner,
	"POST /t/{type}/import":            owner,
	"POST /t/{type}/import/{file}/run": owner,
	"GET /t/{type}/{id}":               people,
	"POST /t/{type}/{id}/delete":       people,
	"GET /t/{type}/{id}/whole":         people,
	"POST /t/{type}/{id}/parts/move":   people,
	"POST /t/{type}/{id}/discard":      people,
	"POST /t/{type}/{id}/props":        people,
	"POST /t/file/upload":              people,
	"GET /files/{id}":                  people,
	"GET /files/{id}/still":            people,
	"POST /speech/get":                 owner,
	"POST /speech/speakers/get":        owner,
	"POST /meetings/teams/connect":     owner,
	"POST /dictate":                    people,
	"GET /api/describe":                people,
	"GET /api/search":                  people,
	"GET /api/describe/{part}":         people,
	"GET /api/describe/{part}/{name}":  people,
	"GET /api/look":                    owner,
	"POST /api/look":                   owner,
	"POST /api/prose":                  people,
	"POST /api/types":                  people,
	"POST /api/types/{type}/fields":    people,
	"POST /api/act/{id}":               people,
	"POST /hook/{token}":               people,
	"POST /api/chat":                   people,
	"POST /api/chat/clear":             people,
	"POST /api/file/upload":            people,
	"POST /api/import/{type}":          owner,
	"GET /api/{type}":                  people,
	"POST /api/{type}":                 people,
	"POST /api/arrange":                people,
	"GET /api/{type}/{id}":             people,
	"PUT /api/{type}/{id}":             people,
	"PATCH /api/{type}/{id}":           people,
	"DELETE /api/{type}/{id}":          people,
	"/api/":                            people,
	"POST /clash/{id}/use":             people,
	"POST /clash/{id}/both":            people,
	"POST /clash/{id}/keep":            people,
	"POST /since/seen":                 people,
	"GET /files/{id}/captions.vtt":     people,
	"GET /files/{id}/sound":            people,
	"GET /files/{id}/sound/{n}":        people,
	"POST /files/{id}/transcribe":      people,
	"GET /files/{id}/transcript.srt":   people,
	"GET /files/{id}/transcript.txt":   people,
	"GET /export/workspace.zip":        owner,
	"GET /export/all.ics":              people,
	"GET /export/{file}":               people,
	"GET /export/{type}/{file}":        people,
}

// routeOf is the route a request goes to, as it was registered: "GET
// /t/{type}", or "" when there is none.
func (s *Server) routeOf(r *http.Request) string {
	h, _ := s.mux.Handler(r)
	if nh, ok := h.(*notFoundHandler); ok {
		_, pattern := nh.mux.Handler(r)
		return pattern
	}
	return ""
}

// ownerOnlyRequest says whether a request is for the owner alone: its
// route says so, or no route says anything, or it reads or writes one of
// the owner's own kinds of record.
func (s *Server) ownerOnlyRequest(r *http.Request) bool {
	pattern := s.routeOf(r)
	if who, said := routeAccess[pattern]; pattern != "" && (!said || who == owner) {
		return true
	}
	return s.ownerOnlyPath(r.URL.Path)
}

// ownerOnlyPath says whether a path reads or writes one of the owner's own
// kinds of record, in a list, a page, the API or an export. An import,
// which writes whatever its columns say, access included, is the owner's
// by its routes.
func (s *Server) ownerOnlyPath(path string) bool {
	for _, typ := range s.app.Types.Types {
		if !typ.Owners {
			continue // the schema says whose each kind is (schema.Type.Owners)
		}
		t := typ.Name
		for _, p := range []string{"/t/" + t, "/api/" + t, "/export/" + t} {
			if path == p || strings.HasPrefix(path, p+"/") || strings.HasPrefix(path, p+".") {
				return true
			}
		}
	}
	return false
}

// RouteAccess is who may use each route, for the test that every route
// says so.
func RouteAccess() map[string]string {
	out := map[string]string{}
	for route, who := range routeAccess {
		out[route] = map[routeFor]string{people: "people", owner: "owner"}[who]
	}
	return out
}
