package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// When two people changed the same text at once on two computers, the
// later version is on the page and the other is kept (store/clash.go).
// The record's page offers it back, with what differs, until someone
// chooses: use it instead, keep both, or keep the page's. Each choice is
// logged and can be undone, so no choice loses anyone's words.

func (s *Server) clashNotices(r *http.Request, t *schema.Type, rec *store.Record) string {
	if _, ok := s.app.Types.Get(store.ClashType); !ok {
		return ""
	}
	clashes, err := s.app.Store.List(store.ClashType, store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return ""
	}
	var b strings.Builder
	back := "/t/" + t.Name + "/" + rec.ID
	for _, c := range clashes {
		if c.Fields["target"] != t.Name || c.Fields["target_id"] != rec.ID || c.Fields["state"] != "open" {
			continue
		}
		field, _ := c.Fields["field"].(string)
		label := field
		if f, ok := t.Field(field); ok {
			label = f.Display()
		}
		who := "on another computer"
		if c.Fields["origin"] == s.app.Store.Origin() {
			who = "on this computer"
		}
		text, _ := c.Fields["text"].(string)
		now, _ := rec.Fields[field].(string)
		base := "/clash/" + c.ID
		props := map[string]any{"id": c.ID, "label": label, "where": who, "text": text,
			"use": base + "/use", "both": base + "/both", "keep": base + "/keep", "from": back}
		if diff := clashParts(now, text); diff != nil {
			props["diff"] = diff
		}
		b.WriteString(string(s.component("clash", props)))
	}
	return b.String()
}

// clashUse puts the other version in place of the page's.
func (s *Server) clashUse(w http.ResponseWriter, r *http.Request) { s.clashChoose(w, r, "used") }

// clashBoth keeps both: the page's, then the other after a blank line.
func (s *Server) clashBoth(w http.ResponseWriter, r *http.Request) { s.clashChoose(w, r, "both") }

// clashKeep keeps the page's, and the other version is set aside.
func (s *Server) clashKeep(w http.ResponseWriter, r *http.Request) { s.clashChoose(w, r, "kept") }

// clashChoose makes one choice and says what it did where the person is,
// with its Undo. Every choice is an update the log can reverse: the text
// goes back as it was, or the offer comes back.
func (s *Server) clashChoose(w http.ResponseWriter, r *http.Request, choice string) {
	r.ParseForm()
	c, err := s.app.Store.Get(store.ClashType, r.PathValue("id"))
	if err != nil || c.Fields["state"] != "open" {
		s.tell(w, r, outcome{Title: "Already chosen", Text: "Someone chose between these versions already."}, "/")
		return
	}
	typ, _ := c.Fields["target"].(string)
	id, _ := c.Fields["target_id"].(string)
	field, _ := c.Fields["field"].(string)
	href := "/t/" + typ + "/" + id
	was, err := s.app.Store.Get(typ, id)
	if err != nil {
		s.failed(w, r, "Not chosen", err, href)
		return
	}
	label, title := field, id
	if t, ok := s.app.Types.Get(typ); ok {
		if f, ok := t.Field(field); ok {
			label = f.Display()
		}
		if tt := strings.TrimSpace(s.title(t, was)); tt != "" {
			title = tt
		}
	}
	if choice == "kept" {
		undo, _, _ := s.apply(r, records.Change{Action: "updated", Component: store.ClashType, ID: c.ID, Detail: label + " of " + title + ", the page's version kept", Href: href},
			records.Op{Type: store.ClashType, ID: c.ID, After: map[string]any{"state": "kept"}})
		s.tell(w, r, outcome{Title: "The page's version is kept", Text: "The other version of " + label + " is set aside.", Undo: undo, Of: "choosing a version of " + label}, href)
		return
	}
	text, _ := c.Fields["text"].(string)
	said, detail := "The other version is in place", " (the other version of "+label+")"
	if choice == "both" {
		cur, _ := was.Fields[field].(string)
		text = strings.TrimRight(cur, "\n") + "\n\n" + text
		said, detail = "Both versions are kept", " (both versions of "+label+")"
	}
	undo, _, err := s.apply(r, records.Change{Action: "updated", Component: typ, ID: id, Detail: title + detail, Href: href},
		records.Op{Type: typ, ID: id, After: map[string]any{field: text}})
	if err != nil {
		s.failed(w, r, "Not chosen", err, href)
		return
	}
	// The offer is used up; undoing the choice puts the text back.
	records.ApplyOps(s.app.Store, records.Op{Type: store.ClashType, ID: c.ID, After: map[string]any{"state": "used"}})
	o := outcome{Title: said, Undo: undo, Of: "choosing a version of " + label}
	if choice == "both" {
		o.Text = label + " has the page's version, then the other. Edit it to join them."
	}
	s.tell(w, r, o, href)
}
