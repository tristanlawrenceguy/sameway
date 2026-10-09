package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Email that came in waits on Today until it is dealt with: made a task,
// or marked sorted. Both are a press, logged, and undone like any change.

// mailToSort are the notes from email not yet dealt with, oldest first.
func (s *Server) mailToSort() []*store.Record {
	if _, ok := s.app.Types.Get("note"); !ok {
		return nil
	}
	recs, _ := s.app.Store.List("note", store.ListOptions{OrderBy: "created_at"})
	var out []*store.Record
	for _, r := range recs {
		if taggedWith(r, toSort) && (taggedWith(r, "email") || taggedWith(r, "shared")) {
			out = append(out, r)
		}
	}
	return out
}

func taggedWith(r *store.Record, tag string) bool {
	tags, _ := r.Fields["tags"].([]any)
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// mailSection is To sort on Today: each email or shared message with what
// the model made of it (triage.go), kept, changed or set aside with a press;
// before it has, or with no model, made a task or set aside by hand.
func (s *Server) mailSection() string {
	notes := s.mailToSort()
	if len(notes) == 0 {
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<h2>To sort</h2><ul class="sw-plain sw-rows">`)
	for _, n := range notes {
		title, _ := n.Fields["title"].(string)
		b.WriteString(`<li class="sw-stack"><a class="sw-link" href="/t/note/` + n.ID + `">` + esc(title) + `</a>`)
		presses := []struct {
			action, label string
			variant       ui.Variant
		}{{"/mail/task", "Make it a task", ui.Secondary}, {"/mail/sorted", "Done with it", ui.Quiet}}
		if sug := s.suggestionFor(n.ID); sug != nil && sug.Task {
			b.WriteString(`<p>` + esc(s.suggestionWords(sug)) + `</p>`) // triage_today.go
			presses = []struct {
				action, label string
				variant       ui.Variant
			}{{"/sort/keep", "Keep", ui.Secondary}, {"/sort/change", "Change", ui.Quiet}, {"/mail/sorted", "Not a task", ui.Quiet}}
		}
		b.WriteString(`<div class="sw-cluster">`)
		for _, f := range presses {
			b.WriteString(string(s.form(ui.Form{Action: f.action, Hidden: ui.Hidden("id", n.ID), Button: &ui.Button{Label: f.label, Context: title, Variant: f.variant}})))
		}
		b.WriteString(`</div></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

// sortedNote takes the to sort tag off a note from email.
func (s *Server) sortedNote(r *http.Request, id string) (*store.Record, string, error) {
	n, err := s.app.Store.Get("note", id)
	if err != nil || !taggedWith(n, toSort) {
		return nil, "", errors.New("that has been dealt with already")
	}
	var tags []any
	for _, t := range n.Fields["tags"].([]any) {
		if t != toSort {
			tags = append(tags, t)
		}
	}
	_, act, err := records.WriteAs(s.app.Store, s.who(r), "updated", "note", id, map[string]any{"tags": tags})
	return n, act, err
}

func (s *Server) mailTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	n, _, err := s.sortedNote(r, r.PostForm.Get("id"))
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	title, _ := n.Fields["title"].(string)
	fields := map[string]any{"title": title}
	if t, ok := s.app.Types.Get("task"); ok {
		if _, ok := t.Field("notes"); ok {
			fields["notes"] = "From the email [" + title + "](/t/note/" + n.ID + ")."
		}
	}
	task, act, err := records.WriteAs(s.app.Store, s.who(r), "created", "task", "", fields)
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	s.tellAt(w, r, outcome{Title: "Task made", Text: title + " is a task now; give it a day on its page.", Undo: act, Of: title}, "/t/task/"+task.ID+"#edit")
}

func (s *Server) mailSorted(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	n, act, err := s.sortedNote(r, r.PostForm.Get("id"))
	if err != nil {
		s.failed(w, r, "Not changed", err, "/today")
		return
	}
	title, _ := n.Fields["title"].(string)
	s.tellAt(w, r, outcome{Title: "Done with", Text: title + " stays in your notes.", Undo: act, Of: title}, "/today")
}
