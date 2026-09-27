package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// When two people changed the same text at once on two computers, the
// later version is on the page and the other is kept (store/clash.go).
// The record's page offers it back, until someone chooses: use it instead,
// or keep what is there.

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
			label = fieldLabel(*f)
		}
		who := "on another computer"
		if c.Fields["origin"] == s.app.Store.Origin() {
			who = "on this computer"
		}
		text, _ := c.Fields["text"].(string)
		props := map[string]any{"id": c.ID, "label": label, "where": who, "text": text,
			"use": "/clash/" + c.ID + "/use", "keep": "/clash/" + c.ID + "/keep", "from": back}
		b.WriteString(string(s.component("clash", props)))
	}
	return b.String()
}

// clashUse puts the other version in place of the one there, as the
// person's change, logged and undoable like any other.
func (s *Server) clashUse(w http.ResponseWriter, r *http.Request) {
	s.clashChoose(w, r, true)
}

// clashKeep keeps what is there, and the other version goes.
func (s *Server) clashKeep(w http.ResponseWriter, r *http.Request) {
	s.clashChoose(w, r, false)
}

func (s *Server) clashChoose(w http.ResponseWriter, r *http.Request, use bool) {
	r.ParseForm()
	back := backTo(r.PostForm.Get("from"))
	c, err := s.app.Store.Get(store.ClashType, r.PathValue("id"))
	if err != nil || c.Fields["state"] != "open" {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	state := "kept"
	if use {
		typ, _ := c.Fields["target"].(string)
		id, _ := c.Fields["target_id"].(string)
		field, _ := c.Fields["field"].(string)
		was, err := s.app.Store.Get(typ, id)
		if err != nil {
			s.failed(w, r, "That version could not be used", err, back)
			return
		}
		if _, err := s.app.Store.Update(typ, id, map[string]any{field: c.Fields["text"]}); err != nil {
			s.failed(w, r, "That version could not be used", err, back)
			return
		}
		s.record(r, chat.Change{Action: "used the other version of", Component: typ, ID: id, Detail: field, Href: "/t/" + typ + "/" + id, Before: was.Fields})
		state = "used"
	}
	s.app.Store.Update(store.ClashType, c.ID, map[string]any{"state": state})
	http.Redirect(w, r, back, http.StatusSeeOther)
}
