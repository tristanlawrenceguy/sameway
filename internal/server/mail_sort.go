package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Email that came in waits on Today until it is dealt with: made a task,
// or marked sorted. Both are a press, logged, and undone like any change.

// toSortItem is an email or a note waiting to be sorted, by id.
func (s *Server) toSortItem(id string) (*store.Record, error) {
	for _, typ := range []string{"email", "note"} {
		if rec, err := s.app.Store.Get(typ, id); err == nil {
			return rec, nil
		}
	}
	return nil, errors.New("that is gone")
}

// mailSection is To sort on Today: each email or shared message with what
// the model made of it (triage.go), kept, changed or set aside with a press;
// before it has, or with no model, made a task or set aside by hand.
func (s *Server) mailSection(list []chat.Waiting) string {
	alone := s.app.Chat.AloneTags() // what needs nothing is folded away
	var nothing []*store.Record
	for _, n := range s.app.Chat.ToSort() {
		if chat.AloneOn(n, alone) != "" {
			nothing = append(nothing, n)
		}
	}
	folded := s.foldedNothing(nothing, alone)
	notes := chat.WaitingOf(list, chat.WaitSort)
	if len(notes) == 0 {
		if folded != "" {
			return `<h2>To sort</h2>` + folded
		}
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<h2>To sort</h2><ul class="sw-plain sw-rows">`)
	tags, other := tagsOn(list), s.otherTags(nil) // tags_today.go: a tag to check sits beside its record
	for _, w := range notes {
		n, title := w.Record, w.Words
		b.WriteString(`<li class="sw-stack"><a class="sw-link" href="` + w.Link + `">` + esc(title) + `</a>`)
		for _, c := range tags[n.Type+"/"+n.ID] {
			b.WriteString(s.tagCheck(c, n, other))
		}
		presses := w.Presses
		if p := suggestedFrom(list, n); p != nil { // suggested.go: what an action made of it
			b.WriteString(`<p>` + esc(p.Words) + `</p>`)
			presses = p.Presses
		}
		b.WriteString(s.presses(presses, title) + `</li>`)
	}
	b.WriteString(`</ul>` + folded)
	return b.String()
}

// sortedNote takes the to sort tag off a note from email.
func (s *Server) sortedNote(r *http.Request, id string) (*store.Record, string, error) {
	n, err := s.toSortItem(id)
	if err != nil || !chat.Tagged(n, toSort) {
		return nil, "", errors.New("that has been dealt with already")
	}
	var tags []any
	for _, t := range n.Fields["tags"].([]any) {
		if t != toSort {
			tags = append(tags, t)
		}
	}
	_, act, err := records.WriteAs(s.app.Store, s.who(r), "updated", n.Type, id, map[string]any{"tags": tags})
	return n, act, err
}

func (s *Server) mailTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	n, _, err := s.sortedNote(r, r.PostForm.Get("id"))
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	title := s.nameOf(n)
	fields := map[string]any{"title": title}
	if t, ok := s.app.Types.Get("task"); ok {
		if _, ok := t.Field("notes"); ok {
			fields["notes"] = "From the email [" + title + "](/t/" + n.Type + "/" + n.ID + ")."
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
	title := s.nameOf(n)
	s.tellAt(w, r, outcome{Title: "Done with", Text: title + " stays in your notes.", Undo: act, Of: title}, "/today")
}

// nameOf is what a record is called, whatever its type.
func (s *Server) nameOf(rec *store.Record) string {
	t, ok := s.app.Types.Get(rec.Type)
	if !ok {
		return rec.ID
	}
	return records.Name(s.app.Store, t, rec)
}

// sortingRefs are the records waiting to be sorted, as type/id.
func (s *Server) sortingRefs() map[string]bool {
	out := map[string]bool{}
	for _, n := range s.app.Chat.ToSort() {
		out[n.Type+"/"+n.ID] = true
	}
	return out
}
