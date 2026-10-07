package server

import (
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// A new version installed by itself ran only from Sameway's next start,
// and a Sameway in the background, opened at sign-in, was not started
// again for weeks. Once one is waiting (update.Pending), the owner's pages
// say so with Restart Sameway, which starts it on the same address and
// stops this one (internal/cli restart.go); the page comes back by itself.

// updateNotice is the version waiting and Restart Sameway, for the owner.
func (s *Server) updateNotice(r *http.Request) template.HTML {
	v := update.Pending()
	if v == "" || s.fleet == nil || s.fleet.Restart == nil || !s.chatFor(r).IsOwner() || r.URL.Path == "/restart" {
		return ""
	}
	form := `<form method="post" action="/restart">` + string(s.component("button", map[string]any{"label": "Restart Sameway", "type": "submit", "variant": "secondary"})) + `</form>`
	return s.component("alert", map[string]any{"kind": "info", "title": "Sameway " + v + " is installed",
		"message": "It runs once Sameway restarts, which takes a few seconds; your workspace is as you left it."}) + template.HTML(form)
}

// restart starts the new version and stops this one; the page it answers
// with waits a moment and goes back to the start.
func (s *Server) restart(w http.ResponseWriter, r *http.Request) {
	if s.fleet == nil || s.fleet.Restart == nil {
		s.failed(w, r, "Not restarted", errNoFleet, "/")
		return
	}
	if err := s.fleet.Restart(); err != nil {
		s.failed(w, r, "Not restarted", err, "/")
		return
	}
	if pageAction(r) {
		tellJSON(w, outcome{Title: "Sameway is restarting", Text: "It is back in a few seconds."}, "/")
		return
	}
	w.Header().Set("Refresh", "6; url=/")
	s.page(w, r, "Sameway is restarting", template.HTML(`<p role="status">Sameway is restarting with the new version. This page goes back to your workspace in a few seconds.</p>`), pageOptions{})
}
