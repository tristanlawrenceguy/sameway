package server

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Every action a person takes from a page ends the same way: they are back
// on the page they were on, and one message says what happened, in the
// same place on every page, once. A failure says what went wrong in plain
// words, and what they typed is not lost: the draft of an edit is kept
// until the edit is said to be saved (18-drafts.js). Before this, each
// action had its own way, or none: a flag left in the address, a message
// in the chat on a page with no chat, a line in the activity log, a bare
// error page. The crew's person-facing findings were mostly those.
// What is said is a web.Outcome.

const outcomeCookie = "sw-outcome"

// tell returns the person to the page they were on, or fallback, with
// the outcome to show there.
func (s *Server) tell(w http.ResponseWriter, r *http.Request, o outcome, fallback string) {
	s.tellAt(w, r, o, web.BackOf(r, fallback))
}

// tellAt sends the person to one page with the outcome: where they were
// is gone, as a deleted record's page is.
func (s *Server) tellAt(w http.ResponseWriter, r *http.Request, o outcome, to string) {
	if o.For == "" && r.Method == http.MethodPost {
		o.For = r.URL.Path
	}
	// A mark saved by its script stays on its page: the outcome comes back
	// as the message itself, for the page to show where it is, and nothing
	// is left in a cookie for the next page to say again (mark/enhance.js).
	if r.Header.Get("X-Requested-With") == "sameway-mark" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, string(s.renderOutcome(o, to)))
		return
	}
	to = web.WithBack(to, web.PlaceOf(r))
	if pageAction(r) {
		tellJSON(w, o, to)
		return
	}
	raw, _ := json.Marshal(o)
	http.SetCookie(w, &http.Cookie{Name: outcomeCookie, Value: base64.RawURLEncoding.EncodeToString(raw),
		Path: "/", MaxAge: 60, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// failed tells a failure where the person is, in words they can act on.
func (s *Server) failed(w http.ResponseWriter, r *http.Request, title string, err error, fallback string) {
	text := ""
	if err != nil {
		text = web.PlainError(err)
	}
	s.tell(w, r, outcome{Failed: true, Title: title, Text: text}, fallback)
}

// told is the outcome waiting for this page, rendered, and gone once
// read: a reload does not say it again.
func (s *Server) told(w http.ResponseWriter, r *http.Request) template.HTML {
	c, err := r.Cookie(outcomeCookie)
	if err != nil {
		return ""
	}
	// A page fetched by the page itself, to follow a change, is not where
	// the person looks for it.
	if r.Header.Get("X-Requested-With") == "sameway-live" {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: outcomeCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	var o outcome
	if err != nil || json.Unmarshal(raw, &o) != nil || o.Title == "" {
		return ""
	}
	return s.renderOutcome(o, r.URL.RequestURI())
}

// renderOutcome is an outcome as the page shows it, its Undo returning to
// from.
func (s *Server) renderOutcome(o outcome, from string) template.HTML {
	kind, state := "success", "done"
	if o.Failed {
		kind, state = "danger", "failed"
	}
	// A short outcome is its title alone, said as the message.
	a := ui.Alert{Kind: ui.AlertKind(kind), Title: o.Title, Message: o.Text, Dismiss: true, Live: true}
	if o.Text == "" {
		a.Title, a.Message = "", o.Title
	}
	alert := string(s.part(a))
	if o.Failed && len(o.Problems) > 0 {
		var items []any
		for _, p := range o.Problems {
			items = append(items, map[string]any{"text": p.Text, "field": p.Field})
		}
		alert = string(s.component("error-summary", map[string]any{"title": o.Title, "items": items}))
	}
	if o.Undo != "" {
		what := o.Of
		if what == "" {
			what = o.Text
		}
		if what == "" {
			what = o.Title
		}
		alert += string(s.form(ui.Form{Action: "/activity/" + o.Undo + "/undo", Class: "sw-outcome__undo", From: from,
			Button: &ui.Button{Label: "Undo", Context: strings.TrimSuffix(what, "."), Variant: ui.Secondary}}))
	}
	return template.HTML(`<div class="sw-outcome" id="outcome" tabindex="-1" data-outcome="` + state + `" data-outcome-for="` + template.HTMLEscapeString(o.For) + `">` + alert + `</div>`)
}
