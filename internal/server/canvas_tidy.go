package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A tab that has gathered what cannot be shown, or the same block twice,
// says so to the person who can change it, with one press that takes them
// away as one change Undo puts back. The assistant is told the same in
// what it reads (chat/layout_clutter.go); the person sees the broken
// blocks on the page, and should not have to ask for each to go. Only
// while there is something to take away.
//
// The line names what the press takes, as Outlook's Sweep and Photos'
// Duplicates do, so a block that is only here twice is not a guess at
// which one goes; and the press takes only what the line named, so a
// page left open never takes something else. A block that shows nothing
// yet is not broken and is never named (design/foundations/layout.md).

// tidyShown is how many blocks the line names before "and N more".
const tidyShown = 3

// tidyOffer is the line under the tabs, or "".
func (s *Server) tidyOffer(r *http.Request, canvas string, blocks []*store.Record) string {
	if a := records.VisitorOf(r.Context()).Access; a == records.View || a == records.Public {
		return ""
	}
	bad := s.app.Chat.Clutter(blocks)
	if len(bad) == 0 {
		return ""
	}
	var ids []string
	for _, b := range blocks {
		if _, ok := bad[b.ID]; ok {
			ids = append(ids, b.ID)
		}
	}
	words := fmt.Sprintf("%d blocks here cannot be shown or repeat another: %s.", len(ids), tidyWhat(blocks, bad))
	if len(ids) == 1 {
		words = "1 block here cannot be shown or repeats another: " + tidyWhat(blocks, bad) + "."
	}
	return `<div class="sw-cluster sw-quiet"><span>` + template.HTMLEscapeString(words) + `</span>` +
		string(s.form(ui.Form{Action: "/tidy", From: records.CanvasPath(canvas), Hidden: ui.Hidden("canvas", canvas, "ids", strings.Join(ids, ",")),
			Button: &ui.Button{Label: "Tidy this page", Variant: ui.Secondary}})) + `</div>`
}

// tidyWhat names each block the press takes, and why, in page order:
// "a button cannot be shown; “Weather” is here twice".
func tidyWhat(recs []*store.Record, bad map[string]string) string {
	var said []string
	n := 0
	for _, b := range recs {
		why, ok := bad[b.ID]
		if !ok {
			continue
		}
		if n++; n > tidyShown {
			continue
		}
		name, _ := b.Fields["component"].(string)
		props, _ := b.Fields["props"].(map[string]any)
		label := "a " + blocks.Noun(name)
		if strings.ContainsAny(label[2:3], "aeiou") {
			label = "an " + label[2:]
		}
		if sum := records.Summarise(name, props); sum != "" {
			label = "“" + sum + "”"
		}
		if strings.HasPrefix(why, "cannot be shown") {
			said = append(said, label+" cannot be shown")
		} else {
			said = append(said, label+" is here twice")
		}
	}
	if n > tidyShown {
		said = append(said, fmt.Sprintf("and %d more", n-tidyShown))
	}
	return strings.Join(said, "; ")
}

// tidyPage takes away a tab's blocks that cannot be shown or repeat
// another, as one change: those the line named that still do, or all of
// them when no names were sent (an agent's POST).
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
	named := map[string]bool{}
	for _, id := range strings.Split(r.PostForm.Get("ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			named[id] = true
		}
	}
	var ops []records.Op
	kept := ""
	for _, b := range blocks {
		if why, ok := bad[b.ID]; ok && (len(named) == 0 || named[b.ID]) {
			ops = append(ops, records.Op{Type: records.BlockType, ID: b.ID})
			if !strings.HasPrefix(why, "cannot be shown") {
				kept = "; the first of each that was here twice stays"
			}
		}
	}
	if len(ops) == 0 {
		s.tellAt(w, r, outcome{Title: "Nothing to tidy", Text: "Every block here can be shown, and none repeats another."}, back)
		return
	}
	n := fmt.Sprintf("%d %s", len(ops), schema.Noun(len(ops), "block"))
	detail := n + " that could not be shown or repeated another"
	act, _, err := face{s}.Apply(r, records.Change{Action: "removed", Component: "block", Detail: detail, Href: back}, ops...)
	if err != nil {
		s.failed(w, r, "Not tidied", err, back)
		return
	}
	gone := n + " are gone from this page"
	if len(ops) == 1 {
		gone = n + " is gone from this page"
	}
	s.tellAt(w, r, outcome{Title: "Tidied", Text: gone + kept + ".", Undo: act, Of: detail}, back)
}
