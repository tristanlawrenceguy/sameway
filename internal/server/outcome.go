package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Every action a person takes from a page ends the same way: they are back
// on the page they were on, and one message says what happened, in the
// same place on every page, once. A failure says what went wrong in plain
// words, and what they typed is not lost: the draft of an edit is kept
// until the edit is said to be saved (16-drafts.js). Before this, each
// action had its own way, or none: a flag left in the address, a message
// in the chat on a page with no chat, a line in the activity log, a bare
// error page. The crew's person-facing findings were mostly those.

// An outcome is what happened, said once on the next page.
type outcome struct {
	// Failed is a failure, announced as an alert; otherwise it is a
	// status, announced politely.
	Failed bool   `json:"f,omitempty"`
	Title  string `json:"t"`
	Text   string `json:"x,omitempty"`
	// For is the form the outcome answers, by its action, so the page
	// knows which draft is now saved and can let it go.
	For string `json:"o,omitempty"`
	// Undo is the activity entry that takes it back, when it can be: the
	// message carries the Undo, where the person is looking.
	Undo string `json:"u,omitempty"`
	// Problems are what stopped a form, each about one field: the message
	// is then an error summary, each problem leading to its field.
	Problems []problem `json:"p,omitempty"`
}

// A problem is one answer a form could not take.
type problem struct {
	Field string `json:"f"`
	Text  string `json:"t"`
}

const outcomeCookie = "sw-outcome"

// tell returns the person to the page they were on, or fallback, with
// the outcome to show there.
func (s *Server) tell(w http.ResponseWriter, r *http.Request, o outcome, fallback string) {
	s.tellAt(w, r, o, backOf(r, fallback))
}

// tellAt sends the person to one page with the outcome: where they were
// is gone, as a deleted record's page is.
func (s *Server) tellAt(w http.ResponseWriter, r *http.Request, o outcome, to string) {
	if o.For == "" && r.Method == http.MethodPost {
		o.For = r.URL.Path
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
		text = plainError(err)
	}
	s.tell(w, r, outcome{Failed: true, Title: title, Text: text}, fallback)
}

// plainError is an error as a person reads it.
func plainError(err error) string {
	if errors.Is(err, store.ErrNotFound) {
		return "It is not there any more; it may have been deleted."
	}
	text := err.Error()
	for _, prefix := range []string{"invalid: ", "validation failed: ", "bad request: "} {
		text = strings.TrimPrefix(text, prefix)
	}
	if text != "" {
		text = strings.ToUpper(text[:1]) + text[1:]
		if !strings.HasSuffix(text, ".") {
			text += "."
		}
	}
	return text
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
	kind, state := "success", "done"
	if o.Failed {
		kind, state = "danger", "failed"
	}
	// A short outcome is its title alone, said as the message.
	props := map[string]any{"kind": kind, "title": o.Title, "message": o.Text, "dismiss": true}
	if o.Text == "" {
		props["title"], props["message"] = "", o.Title
	}
	alert := string(s.component("alert", props))
	if o.Failed && len(o.Problems) > 0 {
		var items []any
		for _, p := range o.Problems {
			items = append(items, map[string]any{"text": p.Text, "field": p.Field})
		}
		alert = string(s.component("error-summary", map[string]any{"title": o.Title, "items": items}))
	}
	if o.Undo != "" {
		what := o.Text
		if what == "" {
			what = o.Title
		}
		alert += `<form method="post" action="/activity/` + template.HTMLEscapeString(o.Undo) + `/undo" class="sw-outcome__undo">` +
			`<input type="hidden" name="from" value="` + template.HTMLEscapeString(r.URL.RequestURI()) + `">` +
			string(s.component("button", map[string]any{"label": "Undo", "context": strings.TrimSuffix(what, "."), "type": "submit", "variant": "secondary"})) + `</form>`
	}
	return template.HTML(`<div class="sw-outcome" id="outcome" tabindex="-1" data-outcome="` + state + `" data-outcome-for="` + template.HTMLEscapeString(o.For) + `">` + alert + `</div>`)
}

// backOf is the page a person acted from: the from the form carries, or
// the page the request came from, on this server; fallback otherwise.
// Only a path here is ever a way back.
func backOf(r *http.Request, fallback string) string {
	if from := r.FormValue("from"); local(from) {
		return from
	}
	if ref, err := url.Parse(r.Referer()); err == nil && ref.Host == r.Host && local(ref.Path) {
		q := ref.Query()
		q.Del("saved")
		ref.RawQuery = q.Encode()
		return ref.RequestURI()
	}
	return fallback
}

// local says a path is a page on this server, not somewhere else.
func local(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.HasPrefix(path, "/\\")
}
