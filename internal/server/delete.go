package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Deleting a record. One step, because it can be taken back: what it was
// goes into the activity log, and the listing the person lands on offers
// to put it back. No page asks "are you sure".

func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	title := s.title(t, rec)
	if err := s.app.Store.Delete(t.Name, rec.ID); err != nil {
		s.fail(w, err)
		return
	}
	// Logged with what it was, so the deletion can be undone.
	s.record(r, chat.Change{Action: "deleted", Component: t.Name, ID: rec.ID, Detail: title, Before: rec.Fields})

	// Render a confirmation page with an alert before redirecting back to
	// the listing, so the person knows the delete actually worked.
	var b strings.Builder
	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "success",
		"title":   "Deleted",
		"message": title + " deleted.",
	})))
	fmt.Fprintf(&b, `<p>%s</p>`, s.component("link", map[string]any{"href": "/t/" + t.Name, "label": "See all " + plural(t.Name), "look": "button"}))
	b.WriteString(`<meta http-equiv="refresh" content="2;url=/t/` + t.Name + `">`)
	s.page(w, r, title+" deleted", template.HTML(b.String()), pageOptions{})
}
