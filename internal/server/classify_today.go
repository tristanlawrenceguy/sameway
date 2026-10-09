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
	alone := s.aloneTags()
	for _, c := range recs {
		if c.Fields["state"] == "suggested" && !alone[strings.ToLower(fmt.Sprint(c.Fields["tag"]))] && len(out) < 20 {
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
	if err != nil || c.Fields["state"] != "suggested" && c.Fields["confirmed_by"] != "judgement" {
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
var starterTags = []struct {
	name, means string
	alone       bool
}{
	{"to do", "Something I have to do: pay, reply, book, bring, send, sign, renew, attend, buy or call. Not newsletters, adverts, receipts for what is paid, or notices that need nothing from me.", false},
	{"important", "Money owed, health, an official or legal deadline (tax, passport, insurance, a lease), school, a work deadline, or someone waiting on my answer. Not plans with friends, errands or reminders of habit.", false},
	{"nothing to do", "Needs nothing from me: newsletters, adverts, receipts, notices and confirmations. Given when no other tag fits.", true},
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
		names = append(names, t.name)
		if !have[t.name] {
			records.WriteAs(s.app.Store, who, "created", chat.TagType, "", map[string]any{"name": t.name, "means": t.means, "alone": t.alone})
		}
	}
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "classify" && a.Fields["what"] == "email" {
			return // set up before; what the person changed since is theirs
		}
	}
	// What comes in, by email or shared from a phone, is tagged, and what is
	// tagged to do is suggested as a task: four actions the person can read,
	// change or remove like any other.
	for _, a := range []map[string]any{
		{"title": "Sort what comes in", "kind": "classify", "when": "added", "what": "email", "tags": names},
		{"title": "Suggest a task for an email to do", "kind": "suggest", "make": "task", "when": "changed", "what": "email", "only": []any{"tags=to do"}},
		{"title": "Sort what is shared", "kind": "classify", "when": "added", "what": "note", "only": []any{"tags=shared"}, "tags": names},
		{"title": "Suggest a task for something shared to do", "kind": "suggest", "make": "task", "when": "changed", "what": "note", "only": []any{"tags=shared", "tags=to do"}},
	} {
		a["check"], a["examples"] = true, 10
		records.WriteAs(s.app.Store, who, "created", records.ActionType, "", a)
	}
}

// aloneTags are the tags given only when no other fits, by name.
func (s *Server) aloneTags() map[string]bool {
	recs, _ := s.app.Store.List(chat.TagType, store.ListOptions{})
	out := map[string]bool{}
	for _, r := range recs {
		if alone, _ := r.Fields["alone"].(bool); alone {
			out[strings.ToLower(fmt.Sprint(r.Fields["name"]))] = true
		}
	}
	return out
}

// hasAlone is the tag a record has that is given only when no other fits.
func (s *Server) hasAlone(rec *store.Record, alone map[string]bool) string {
	for _, t := range records.StringList(rec.Fields["tags"]) {
		if alone[strings.ToLower(t)] {
			return t
		}
	}
	return ""
}

// latestTag is the newest classification of a tag on a record, to change.
func (s *Server) latestTag(rec *store.Record, tag string) *store.Record {
	cls, _ := s.app.Store.List(chat.ClassificationType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 200})
	for _, c := range cls {
		if c.Fields["record"] == rec.Type+"/"+rec.ID && strings.EqualFold(fmt.Sprint(c.Fields["tag"]), tag) {
			return c
		}
	}
	return nil
}

// foldedNothing is what was looked at and needs nothing, folded away as a
// count on Today: newsletters and receipts do not compete for attention,
// and one changed to another tag teaches the action what it missed.
func (s *Server) foldedNothing(items []*store.Record, alone map[string]bool) string {
	esc := template.HTMLEscapeString
	var other []any
	for _, t := range s.tagNames() {
		if !alone[strings.ToLower(t)] {
			other = append(other, t)
		}
	}
	var b strings.Builder
	n, tag := 0, ""
	for _, rec := range items {
		t := s.hasAlone(rec, alone)
		c := s.latestTag(rec, t)
		if t == "" || c == nil || len(other) == 0 {
			continue
		}
		n, tag = n+1, t
		name := s.nameOf(rec)
		why, _ := c.Fields["why"].(string)
		b.WriteString(`<li class="sw-stack"><p><a class="sw-link" href="/t/` + rec.Type + `/` + rec.ID + `">` + esc(name) + `</a>`)
		if why != "" && why != "no other tag fits" {
			b.WriteString(`: ` + esc(why))
		}
		b.WriteString(`</p><form method="post" action="/tags/change" class="sw-cluster"><input type="hidden" name="id" value="` + c.ID + `">` +
			string(s.component("select", map[string]any{"label": "Change to", "context": name, "name": "to", "as": "dropdown", "options": other, "value": other[0]})) +
			string(s.component("button", map[string]any{"label": "Change", "context": name, "type": "submit", "variant": "quiet"})) + `</form></li>`)
	}
	if n == 0 {
		return ""
	}
	body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": strings.ToUpper(tag[:1]) + tag[1:], "count": n, "of": "item"}, template.HTML(`<ul class="sw-plain sw-rows">`+b.String()+`</ul>`))
	if err != nil {
		return ""
	}
	return string(body)
}
