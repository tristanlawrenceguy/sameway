package server

import (
	"errors"
	"html/template"
	"net/http"
	"runtime"
)

var errNoFleet = errors.New("this Sameway was not started to be stopped from its page; stop it where it was started")

// Sameway runs in the background once double-clicked (internal/cli
// apart.go), and on a Mac as an app with no window, so a person stops it
// here: Quit Sameway on the Workspaces page. It used to be closing a black
// window, and on a Mac there was no way at all short of the Activity
// Monitor.

// quitSection is Quit Sameway, for the owner, under This workspace.
func (s *Server) quitSection() string {
	if s.fleet == nil || s.fleet.Exit == nil {
		return ""
	}
	return `<form method="post" action="/quit" class="sw-stack">` +
		string(s.component("button", map[string]any{"label": "Quit Sameway", "type": "submit", "variant": "secondary"})) +
		`<p class="sw-small sw-muted">Stops Sameway on this computer until you open it again ` + reopenWhere() + `. Reminders do not ring while it is stopped.</p></form>`
}

// reopenWhere is where a person opens Sameway again on this computer.
func reopenWhere() string {
	switch runtime.GOOS {
	case "windows":
		return "from the Start menu"
	case "darwin":
		return "from Applications"
	}
	return "the way you opened it"
}

// quit stops this Sameway after saying so.
func (s *Server) quit(w http.ResponseWriter, r *http.Request) {
	if s.fleet == nil || s.fleet.Exit == nil {
		s.failed(w, r, "Not stopped", errNoFleet, "/workspaces")
		return
	}
	said := "Open it again " + reopenWhere() + "."
	if pageAction(r) {
		tellJSON(w, outcome{Title: "Sameway has stopped", Text: said}, "")
	} else {
		s.page(w, r, "Sameway has stopped", template.HTML(`<p>`+template.HTMLEscapeString(said)+` Your workspace is as you left it.</p>`), pageOptions{})
	}
	s.fleet.Exit() // a moment after the page is written: open.go
}

// quitRoutes are Quit Sameway and opening it at sign-in (at_login.go).
func (s *Server) quitRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /quit", s.quit)
	m.HandleFunc("POST /at-login", s.atLoginSet)
}
