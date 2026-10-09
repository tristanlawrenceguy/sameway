package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A tag a classify action gave waits on Today as a suggestion: kept with a
// press, taken off, or changed to another tag. What the person did is what
// the action follows next time (chat/judgement.go), so each press is their
// judgement taught once, not a chore repeated.

// tagsToCheck are the suggested tags, newest first, a few at a time.
func (s *Server) tagsToCheck() []*store.Record {
	if _, ok := s.app.Types.Get(chat.ClassificationType); !ok {
		return nil
	}
	recs, _ := s.app.Store.List(chat.ClassificationType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 100})
	var out []*store.Record
	for _, c := range recs {
		if c.Fields["state"] == "suggested" && len(out) < 20 {
			out = append(out, c)
		}
	}
	return out
}

// taggedRecord is the record a classification is about.
func (s *Server) taggedRecord(c *store.Record) (*store.Record, error) {
	typ, id, _ := strings.Cut(fmt.Sprint(c.Fields["record"]), "/")
	return s.app.Store.Get(typ, id)
}

// tagSection is Tags to check on Today.
func (s *Server) tagSection() string {
	cls := s.tagsToCheck()
	if len(cls) == 0 {
		return ""
	}
	esc := template.HTMLEscapeString
	var other []any
	for _, t := range s.tagNames() {
		other = append(other, t)
	}
	var b strings.Builder
	b.WriteString(`<h2>Tags to check</h2><ul class="sw-plain sw-rows">`)
	for _, c := range cls {
		rec, err := s.taggedRecord(c)
		if err != nil {
			continue
		}
		name, tag := s.nameOf(rec), fmt.Sprint(c.Fields["tag"])
		why, _ := c.Fields["why"].(string)
		b.WriteString(`<li class="sw-stack"><p><a class="sw-link" href="/t/` + rec.Type + `/` + rec.ID + `">` + esc(name) + `</a>: tagged <strong>` + esc(tag) + `</strong>`)
		if why != "" {
			b.WriteString(`, ` + esc(strings.TrimSuffix(why, ".")))
		}
		b.WriteString(`.</p><div class="sw-cluster">`)
		hidden := `<input type="hidden" name="id" value="` + c.ID + `">`
		context := tag + " on " + name
		b.WriteString(`<form method="post" action="/tags/keep">` + hidden + string(s.component("button", map[string]any{"label": "Keep", "context": context, "type": "submit", "variant": "secondary"})) + `</form>`)
		b.WriteString(`<form method="post" action="/tags/off">` + hidden + string(s.component("button", map[string]any{"label": "Take it off", "context": context, "type": "submit", "variant": "quiet"})) + `</form>`)
		if len(other) > 1 {
			b.WriteString(`<form method="post" action="/tags/change" class="sw-cluster">` + hidden +
				string(s.component("select", map[string]any{"label": "Change to", "context": context, "name": "to", "as": "dropdown", "options": other, "value": tag})) +
				string(s.component("button", map[string]any{"label": "Change", "context": context, "type": "submit", "variant": "quiet"})) + `</form>`)
		}
		b.WriteString(`</div></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

func (s *Server) tagNames() []string {
	recs, _ := s.app.Store.List(chat.TagType, store.ListOptions{OrderBy: "name"})
	var out []string
	for _, r := range recs {
		if n, _ := r.Fields["name"].(string); strings.TrimSpace(n) != "" {
			out = append(out, n)
		}
	}
	return out
}

// suggestion is the classification a press is about, still suggested.
func (s *Server) suggestion(r *http.Request) (*store.Record, *store.Record, error) {
	r.ParseForm()
	c, err := s.app.Store.Get(chat.ClassificationType, r.PostForm.Get("id"))
	if err != nil || c.Fields["state"] != "suggested" {
		return nil, nil, errors.New("that tag has been dealt with already")
	}
	rec, err := s.taggedRecord(c)
	if err != nil {
		return nil, nil, errors.New("what it tagged is gone")
	}
	return c, rec, nil
}

func (s *Server) tagKeep(w http.ResponseWriter, r *http.Request) {
	c, rec, err := s.suggestion(r)
	if err != nil {
		s.failed(w, r, "Not kept", err, "/today")
		return
	}
	records.ApplyOps(s.app.Store, records.Op{Type: chat.ClassificationType, ID: c.ID, After: map[string]any{"state": "confirmed", "confirmed_by": "person"}})
	s.tellAt(w, r, outcome{Title: "Kept", Text: fmt.Sprintf("%s keeps the tag %s; the action follows that next time.", s.nameOf(rec), c.Fields["tag"])}, "/today")
}

// retag takes a tag off a record and puts another on, or none: the record
// changing is what marks the suggestion taken off or changed (judgement.go).
func (s *Server) retag(r *http.Request, rec *store.Record, off, on string) (string, error) {
	var tags []any
	for _, t := range records.StringList(rec.Fields["tags"]) {
		if !strings.EqualFold(t, off) && !strings.EqualFold(t, on) {
			tags = append(tags, t)
		}
	}
	if on != "" {
		tags = append(tags, on)
	}
	_, act, err := records.WriteAs(s.app.Store, s.who(r), "updated", rec.Type, rec.ID, map[string]any{"tags": tags})
	return act, err
}

func (s *Server) tagOff(w http.ResponseWriter, r *http.Request) {
	c, rec, err := s.suggestion(r)
	if err == nil {
		var act string
		if act, err = s.retag(r, rec, fmt.Sprint(c.Fields["tag"]), ""); err == nil {
			s.tellAt(w, r, outcome{Title: "Taken off", Text: fmt.Sprintf("%s no longer has the tag %s.", s.nameOf(rec), c.Fields["tag"]), Undo: act}, "/today")
			return
		}
	}
	s.failed(w, r, "Not changed", err, "/today")
}

func (s *Server) tagChange(w http.ResponseWriter, r *http.Request) {
	c, rec, err := s.suggestion(r)
	to := strings.TrimSpace(r.PostForm.Get("to"))
	if err == nil && strings.EqualFold(to, fmt.Sprint(c.Fields["tag"])) {
		s.tagKeep(w, r) // the same tag chosen is the tag kept
		return
	}
	if err == nil {
		var act string
		if act, err = s.retag(r, rec, fmt.Sprint(c.Fields["tag"]), to); err == nil {
			s.tellAt(w, r, outcome{Title: "Changed", Text: fmt.Sprintf("%s is tagged %s instead of %s.", s.nameOf(rec), to, c.Fields["tag"]), Undo: act}, "/today")
			return
		}
	}
	s.failed(w, r, "Not changed", err, "/today")
}

// starterTags are the tags sorting begins with; the person changes what
// each means, or adds their own, on the tags' page.
var starterTags = [][2]string{
	{"to do", "Something I have to do: pay, reply, book, bring, send, sign, renew, attend, buy or call."},
	{"important", "Money owed, health, an official or legal deadline (tax, passport, insurance), school, a work deadline, or someone waiting on my answer."},
}

// setUpSorting, the first time email is connected, makes the starter tags
// and the action that gives them to each email that comes in, checked
// against the person's past choices. Both are ordinary records: the person
// changes the tags, the meanings, or the action, or removes them.
func (s *Server) setUpSorting() {
	if _, ok := s.app.Types.Get(chat.TagType); !ok {
		return
	}
	if _, ok := s.app.Types.Get("email"); !ok {
		return
	}
	who := records.Who{Actor: "system", Via: "setting up email"}
	have := map[string]bool{}
	for _, n := range s.tagNames() {
		have[strings.ToLower(n)] = true
	}
	var names []any
	for _, t := range starterTags {
		names = append(names, t[0])
		if !have[t[0]] {
			records.WriteAs(s.app.Store, who, "created", chat.TagType, "", map[string]any{"name": t[0], "means": t[1]})
		}
	}
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "classify" && a.Fields["what"] == "email" {
			return
		}
	}
	records.WriteAs(s.app.Store, who, "created", records.ActionType, "", map[string]any{
		"title": "Sort what comes in", "kind": "classify", "when": "added", "what": "email", "tags": names, "check": true, "examples": 10})
}
