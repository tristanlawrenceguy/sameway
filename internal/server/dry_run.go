package server

import (
	"net/http"
)

// A change sent with Sameway-Dry-Run: 1 is made on a throwaway copy of the
// workspace (app.Sandbox) and answered as it would be here, so an agent
// finds out whether it would be refused, and how, without making it: the
// same handler on the same records, so the same checks and the same
// words. What reaches beyond the workspace is not tried, since a copy
// cannot hold it back: each route in the table says which it is (reach),
// and one that says nothing is not tried. It was a list of address
// prefixes, which missed running an action from the API, sending to a
// phone, the cloud folder, calendars, mail and the phone's pairing.

// DryRun is the header that asks for it, and is answered with what was done.
const DryRun = "Sameway-Dry-Run"

// reach says where what a route changes is, so whether a copy can try it.
type reach int

const (
	// unreached is a route nobody said anything about: not tried.
	unreached reach = iota
	// inward changes only this workspace's own records, files and settings.
	inward
	// outward reaches beyond them: the network, a notification, a
	// command, another workspace, this computer, the assistant's model,
	// another computer, or work that goes on after the answer (writing a
	// recording down), on a copy that is gone by then.
	outward
	// byTool is /api/tools/{name}: the tool it names says, by its
	// Traits.OpenWorld (chat/op.go).
	byTool
)

// tried answers a dry run, and says whether the request was one.
func (s *Server) tried(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(DryRun) == "" || r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	if !s.mayTry(r) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "cannot_try",
			Message: r.URL.Path + " reaches outside the workspace, so a copy cannot try it; send it without " + DryRun + " when the person has said yes"}})
		return true
	}
	sb, done, err := s.app.Sandbox()
	if err != nil {
		writeError(w, err)
		return true
	}
	defer done()
	w.Header().Set(DryRun, "tried on a copy; nothing was changed")
	there := New(sb)
	there.serve(w, r)
	return true
}

// mayTry says whether a copy can try a request: its route stays inside the
// workspace. A request no route serves is answered not found, there as
// here.
func (s *Server) mayTry(r *http.Request) bool {
	pattern := s.routeOf(r)
	if pattern == "" {
		return true
	}
	rt, _ := routeAt(pattern)
	switch rt.reach {
	case inward:
		return true
	case byTool:
		return !toolOutward(r.URL.Path)
	}
	return false
}
