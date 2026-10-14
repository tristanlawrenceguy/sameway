package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A tab that has gathered what cannot be shown, or the same block twice,
// says so to the person who can change it, with one press that takes them
// away as one change Undo puts back. The assistant is told the same in
// what it reads (chat/layout_clutter.go); the person sees the broken
// blocks on the page, and should not have to ask for each to go. Only
// while there is something to take away.

// tidyOffer is the line under the tabs, or "".
func (s *Server) tidyOffer(r *http.Request, canvas string, blocks []*store.Record) string {
	if a := records.VisitorOf(r.Context()).Access; a == records.View || a == records.Public {
		return ""
	}
	bad := s.app.Chat.Clutter(blocks)
	if len(bad) == 0 {
		return ""
	}
	words := fmt.Sprintf("%d blocks here cannot be shown or repeat another.", len(bad))
	if len(bad) == 1 {
		words = "1 block here cannot be shown or repeats another."
	}
	return `<div class="sw-cluster sw-quiet"><span>` + template.HTMLEscapeString(words) + `</span>` +
		string(s.form(ui.Form{Action: "/tidy", From: records.CanvasPath(canvas), Hidden: ui.Hidden("canvas", canvas),
			Button: &ui.Button{Label: "Tidy this page", Variant: ui.Secondary}})) + `</div>`
}

// tidyPage takes away a tab's blocks that cannot be shown or repeat
// another, as one change.
func (s *Server) tidyPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	canvas := r.PostForm.Get("canvas")
	back := records.CanvasPath(canvas)
	all, err := s.app.Store.List(records.BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		s.failed(w, r, "Not tidied", err, back)
		return
	}
	blocks := records.OnCanvas(all, canvas)
	bad := s.app.Chat.Clutter(blocks)
	var ops []records.Op
	for _, b := range blocks {
		if _, ok := bad[b.ID]; ok {
			ops = append(ops, records.Op{Type: records.BlockType, ID: b.ID})
		}
	}
	if len(ops) == 0 {
		s.tellAt(w, r, outcome{Title: "Nothing to tidy", Text: "Every block here can be shown, and none repeats another."}, back)
		return
	}
	detail := fmt.Sprintf("%d blocks that could not be shown or repeated another", len(ops))
	act, _, err := face{s}.Apply(r, records.Change{Action: "removed", Component: "block", Detail: detail, Href: back}, ops...)
	if err != nil {
		s.failed(w, r, "Not tidied", err, back)
		return
	}
	s.tellAt(w, r, outcome{Title: "Tidied", Text: fmt.Sprintf("%d blocks are gone from this page.", len(ops)), Undo: act, Of: detail}, back)
}
