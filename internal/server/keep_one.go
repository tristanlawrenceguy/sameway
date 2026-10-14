package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Records nothing tells apart (records.Twins), said under a list's count
// with one press to keep the newest. The list already told them apart by
// when each was added, "the first of 2"; this is the same finding, offered
// as what to do about it. It is there only while they are, and the press
// is one change that Undo takes back (records.KeepOne, as keep_one does).
// If they are really two, Rename one opens the oldest to be named apart:
// a name of its own tells it apart for everyone, where a hidden "not the
// same" would leave two controls of one name (design/foundations/names.md).

// twinsShown is how many groups of them a list offers at once.
const twinsShown = 3

// listLede is a list's count, and under it what in it is one thing twice.
func (s *Server) listLede(t *schema.Type, recs []*store.Record) template.HTML {
	lede := howMany(t, recs)
	twins := records.Twins(s.app.Store, t)
	for _, g := range twins[:min(twinsShown, len(twins))] {
		name := s.title(t, g[0])
		words := fmt.Sprintf("%s called %s, and nothing tells them apart.", capitalize(schema.Count(len(g), t.Name)), name)
		lede += template.HTML(`<div class="sw-lede sw-cluster sw-quiet"><span>`+template.HTMLEscapeString(words)+`</span>`) +
			s.form(ui.Form{Action: "/t/" + t.Name + "/" + g[0].ID + "/keep-one", From: "/t/" + t.Name,
				Button: &ui.Button{Label: "Keep the newest", Context: name, Variant: ui.Secondary}}) +
			s.part(ui.Link{Href: "/t/" + t.Name + "/" + g[len(g)-1].ID + "#edit", Label: "Rename one", Context: name, Look: ui.LookButton}) + `</div>`
	}
	return lede
}

// keepOneForm keeps a record and merges into it the others nothing tells
// apart from it.
func (s *Server) keepOneForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	list := "/t/" + t.Name
	kept, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Not merged", err, list)
		return
	}
	others := records.TwinsOf(s.app.Store, t, kept)
	if len(others) == 0 {
		s.tellAt(w, r, outcome{Title: "Nothing to merge", Text: "Nothing else is called " + s.title(t, kept) + " now."}, list)
		return
	}
	name := s.title(t, kept)
	detail := fmt.Sprintf("%s, %d into one", name, len(others)+1)
	act, _, err := face{s}.Apply(r, records.Change{Action: "merged", Component: t.Name, ID: kept.ID, Detail: detail}, records.KeepOne(s.app.Store, t, kept, others)...)
	if err != nil {
		s.failed(w, r, "Not merged", err, list)
		return
	}
	s.tellAt(w, r, outcome{Title: "Merged", Text: fmt.Sprintf("%d %s called %s are one now.", len(others)+1, schema.Plural(t.Name), name), Undo: act, Of: detail}, web.BackOf(r, list))
}
