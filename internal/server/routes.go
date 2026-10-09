package server

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Every address the server answers is one table, by area across the
// routes_*.go files: what serves it, who may use it, for a page action the
// assistant's tool that does the same or why it is a person's alone, and
// whether the internet may read it when what it shows is published. The
// mux, who may make a request (access_routes.go), the page actions' tools
// and what is public (publish.go) are all read from here; they were four
// lists kept in step by tests.

// route is one address.
type route struct {
	pattern string
	handle  func(*Server, http.ResponseWriter, *http.Request)
	access  routeFor
	// tool is a page action's op (chat/op.go): what the assistant does to
	// do the same. persons is instead why it is a person's alone.
	tool, persons string
	// public says the internet may read it, when what it shows is
	// published: a published tab, list or record, its files and styles.
	public bool
	// reach is where what it changes is, so whether a dry run may try it
	// on a copy (dry_run.go). Every route but a GET says.
	reach reach
}

// routeTable is every route. It is put together in init, because what a
// route serves can lead back to the table (a dry run serves a copy).
var routeTable []route

var routeIndex = map[string]int{}

func init() {
	for _, area := range [][]route{pageRoutes, recordRoutes, ownerRoutes, apiRoutes} {
		routeTable = append(routeTable, area...)
	}
	for i, rt := range routeTable {
		if _, twice := routeIndex[rt.pattern]; twice {
			panic("two routes are " + rt.pattern)
		}
		if _, ok := chat.OpFor(rt.tool); rt.tool != "" && !ok {
			panic(rt.pattern + " names the tool " + rt.tool + ", which the assistant does not have")
		}
		routeIndex[rt.pattern] = i
	}
}

// routes registers every route in the table.
func (s *Server) routes() {
	for _, rt := range routeTable {
		handle := rt.handle
		s.mux.HandleFunc(rt.pattern, func(w http.ResponseWriter, r *http.Request) { handle(s, w, r) })
	}
}

// plain is a handler that needs no server, as a route's.
func plain(h http.HandlerFunc) func(*Server, http.ResponseWriter, *http.Request) {
	return func(_ *Server, w http.ResponseWriter, r *http.Request) { h(w, r) }
}

// pageRoutes are the canvas, the conversation, and the pages around them.
var pageRoutes = []route{
	{pattern: "GET /{$}", handle: (*Server).canvasPage, access: people, public: true},
	{pattern: "GET /c/{canvas}", handle: (*Server).canvasPage, access: people, public: true},
	{pattern: "GET /canvas/{id}", handle: (*Server).focusPage, access: people},
	{pattern: "POST /canvas/{id}/props", handle: (*Server).blockProps, access: people, tool: "update_component", reach: inward},
	{pattern: "POST /canvas/{id}/keep", handle: (*Server).canvasKeep, access: people, tool: "update_component", reach: inward},
	{pattern: "POST /canvas/{id}/delete", handle: (*Server).canvasDelete, access: people, tool: "remove_component", reach: inward},
	// And only those who may change it: measure.go.
	{pattern: "POST /canvas/measure", handle: (*Server).measurePost, access: people, persons: "their browser says how the page came out on their screen; the assistant reads it in Layout now", reach: inward},
	{pattern: "GET /chat", handle: (*Server).chatPage, access: people},
	{pattern: "POST /chat", handle: (*Server).chatSend, access: people, persons: "it is what they say to the assistant", reach: outward},
	{pattern: "POST /chat/stream", handle: (*Server).chatStream, access: people, persons: "the same", reach: outward},
	{pattern: "POST /chat/stop", handle: (*Server).chatStop, access: people, persons: "stopping the assistant", reach: inward},
	{pattern: "GET /chat/live", handle: (*Server).chatLive, access: people},
	{pattern: "POST /chat/clear", handle: (*Server).chatClear, access: people, tool: "clear_conversation", reach: inward},
	{pattern: "POST /chat/new", handle: (*Server).chatNew, access: people, persons: "which conversation they are in is theirs", reach: inward},
	{pattern: "POST /chat/open", handle: (*Server).chatOpen, access: people, persons: "the same", reach: inward},
	{pattern: "POST /chat/delete", handle: (*Server).chatDelete, access: people, persons: "the same", reach: inward},
	{pattern: "POST /proposal/{id}/accept", handle: (*Server).proposalAccept, access: owner, persons: "the answer to the assistant's own question", reach: outward},
	{pattern: "POST /proposal/{id}/dismiss", handle: (*Server).proposalDismiss, access: owner, persons: "the same", reach: inward},
	{pattern: "POST /proposal/{id}/instead", handle: (*Server).proposalInstead, access: owner, persons: "the same", reach: inward},
	{pattern: "GET /activity", handle: (*Server).activityPage, access: owner},
	{pattern: "POST /activity/{id}/undo", handle: (*Server).undo, access: owner, tool: "undo_change", reach: inward},
	{pattern: "GET /help", handle: (*Server).helpPage, access: people},
	{pattern: "POST /help/set", handle: (*Server).helpSet, access: owner, tool: "set_setting", reach: inward},
	{pattern: "POST /clock/set", handle: (*Server).clockSet, access: people, tool: "create_record", reach: inward},
	{pattern: "POST /clock/{id}/done", handle: (*Server).clockDone, access: people, tool: "update_record", reach: inward},
	{pattern: "POST /clock/{id}/snooze", handle: (*Server).clockSnooze, access: people, tool: "update_record", reach: inward},
	{pattern: "POST /habit/{id}/log", handle: (*Server).habitLog, access: people, tool: "create_record", reach: inward},
	{pattern: "GET /events", handle: (*Server).events, access: people},
	{pattern: "GET /search", handle: (*Server).searchPage, access: people},
	{pattern: "GET /when", handle: (*Server).whenRead, access: people},
	{pattern: "POST /dictate", handle: (*Server).dictate, access: people, persons: "their voice", reach: inward},
	{pattern: "POST /sync", handle: (*Server).syncExchange, access: people, persons: "computers exchanging changes, not a change", reach: outward},
	// Working together: choosing between two versions written at once
	// (clash.go), and saying they have caught up on what others did (since.go).
	{pattern: "POST /clash/{id}/use", handle: (*Server).clashUse, access: people, persons: "which of two versions to keep", reach: inward},
	{pattern: "POST /clash/{id}/both", handle: (*Server).clashBoth, access: people, persons: "the same", reach: inward},
	{pattern: "POST /clash/{id}/keep", handle: (*Server).clashKeep, access: people, persons: "the same", reach: inward},
	{pattern: "POST /since/seen", handle: (*Server).sinceSeen, access: people, persons: "what they have seen", reach: inward},
	{pattern: "GET /design", handle: (*Server).designPage, access: people},
	{pattern: "GET /design/sameway.css", handle: (*Server).stylesheet, access: people, public: true},
	{pattern: "GET /design/sameway.js", handle: (*Server).script, access: people, public: true},
	{pattern: "GET /design/base/{file}", handle: (*Server).baseFile, access: people, public: true},
	// The icons: icon.go.
	{pattern: "GET /favicon.svg", handle: plain(icon("icon.svg", "image/svg+xml")), access: people},
	{pattern: "GET /favicon.ico", handle: plain(icon("icon.ico", "image/x-icon")), access: people},
	{pattern: "GET /icon-192.png", handle: plain(icon("icon-192.png", "image/png")), access: people},
	{pattern: "GET /icon-512.png", handle: plain(icon("icon-512.png", "image/png")), access: people},
	{pattern: "GET /icon-square-180.png", handle: plain(icon("icon-square-180.png", "image/png")), access: people},
	{pattern: "GET /icon-square-512.png", handle: plain(icon("icon-square-512.png", "image/png")), access: people},
	{pattern: "GET /manifest.webmanifest", handle: plain(appManifest), access: people},
}
