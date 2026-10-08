package server

import (
	"net/http"
	"strings"
)

// A change sent with Sameway-Dry-Run: 1 is made on a throwaway copy of the
// workspace (app.Sandbox) and answered as it would be here, so an agent
// finds out whether it would be refused, and how, without making it: the
// same handler on the same records, so the same checks and the same
// words. What reaches beyond the workspace (an action, a hook, another
// workspace) is not tried, since a copy cannot hold it back.

// DryRun is the header that asks for it, and is answered with what was done.
const DryRun = "Sameway-Dry-Run"

// outward are the routes a copy cannot try.
var outward = []string{"/act/", "/hook/", "/workspaces/"}

// tried answers a dry run, and says whether the request was one.
func (s *Server) tried(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(DryRun) == "" || r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	for _, p := range outward {
		if strings.HasPrefix(r.URL.Path, p) || toolOutward(r.URL.Path) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "cannot_try",
				Message: r.URL.Path + " reaches outside the workspace, so a copy cannot try it; send it without " + DryRun + " when the person has said yes"}})
			return true
		}
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
