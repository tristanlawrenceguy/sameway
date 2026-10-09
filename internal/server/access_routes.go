package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Who may use each route, said for every route in the route table
// (routes.go). It used to be a list of address prefixes that were the
// owner's, with everything else open to anyone let in, so a route whose
// address matched no prefix was shared by default: GET /api/workspaces,
// the other workspaces on this machine, was one until it was added to the
// list by hand. Now a route nobody said anything about is the owner's
// alone, and TestEveryRouteSaysWhoMayUseIt fails until it is said.

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
	if rt, said := routeAt(pattern); pattern != "" && (!said || rt.access != people) {
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

// routeAt is a route by its pattern, and whether there is one.
func routeAt(pattern string) (route, bool) {
	i, ok := routeIndex[pattern]
	if !ok {
		return route{}, false
	}
	return routeTable[i], true
}

// Route is one route as the tests read it: who may use it ("people",
// "owner", or "" when nobody said), and for a page action the tool that
// does the same or why it is a person's, and where what it changes is
// ("inward", "outward", "tool", or "" when nobody said; dry_run.go).
type Route struct {
	Pattern, Access, Tool, Persons, Reach string
	Public                                bool
}

// Routes is every route, for the tests that every one says who may use
// it and every page action what the assistant does instead.
func Routes() []Route {
	var out []Route
	for _, rt := range routeTable {
		out = append(out, Route{Pattern: rt.pattern, Access: map[web.Access]string{people: "people", owner: "owner"}[rt.access],
			Tool: rt.tool, Persons: rt.persons, Public: rt.public, Reach: map[reach]string{inward: "inward", outward: "outward", web.ByTool: "tool"}[rt.reach]})
	}
	return out
}
