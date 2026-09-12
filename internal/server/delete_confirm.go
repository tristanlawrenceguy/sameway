package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// confirmDeletePage shows a confirmation screen before destroying a record.
func (s *Server) confirmDeletePage(w http.ResponseWriter, r *http.Request) {
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

	var b strings.Builder
	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "danger",
		"title":   fmt.Sprintf("Delete %s?", t.Name),
		"message": "This cannot be undone.",
	})))
	fmt.Fprintf(&b, `<form method="post" action="/t/%s/%s/delete">%s</form>`,
		t.Name, rec.ID, s.component("button", map[string]any{
			"label": "Confirm deletion", "type": "submit", "variant": "danger"}))
	b.WriteString(string(s.component("link", map[string]any{
		"href": "/t/" + t.Name + "/" + rec.ID, "label": "Cancel"})))

	s.page(w, r, "Delete "+titleOf(t, rec), template.HTML(b.String()), pageOptions{})
}
