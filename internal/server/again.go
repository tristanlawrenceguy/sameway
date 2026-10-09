package server

import (
	"html/template"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A turn that failed, with a provider busy or limiting for a moment, left
// the person to type their message again. The newest failed reply now
// carries Send again, which sends what they asked as it was. It is a form,
// so it works without a script.

// messageShown is a message as the page shows it: the message, and Send again
// under the newest one when it is a failure.
func (s *Server) messageShown(m *store.Record, from string, latest bool) template.HTML {
	out := s.component("message", s.messageProps(m, from, latest))
	if !latest || m.Fields["role"] != "error" {
		return out
	}
	asked := s.askedBefore(m)
	if asked == "" {
		return out
	}
	return out + s.form(ui.Form{Action: "/chat", Class: "sw-again", From: from, Hidden: ui.Hidden("message", asked),
		Button: &ui.Button{Label: "Send again", Variant: ui.Secondary}})
}

// askedBefore is what the person said last before a message, in the same
// chat.
func (s *Server) askedBefore(m *store.Record) string {
	recent, _ := s.app.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 40})
	for _, r := range recent {
		if r.Fields["conversation"] != m.Fields["conversation"] || r.CreatedAt.After(m.CreatedAt) || r.Fields["role"] != "user" {
			continue
		}
		said, _ := r.Fields["content"].(string)
		return said
	}
	return ""
}
