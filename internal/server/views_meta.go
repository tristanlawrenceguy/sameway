package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The few words a page says about a record beside its title: its box, its
// state, the day that matters to it, what it belongs to, when it was made.

// lede is the line under a record's title: its box when it has one, its
// facts as chips, then when it was made. On the record's own page the box
// is labelled with its word, Done or Pinned, where it can be seen; the
// record is the page's heading, so the label does not say it again, and the
// box says checked, so no chip says the state a second time.
func (s *Server) lede(r *http.Request, t *schema.Type, rec *store.Record) template.HTML {
	box := ""
	if props, ok := s.markOf(t, rec); ok {
		delete(props, "context")
		box = string(s.component("mark", props))
	}
	// A div, not a p: a form inside a p ends the p, and the line came apart.
	return template.HTML(`<div class="sw-lede">` + box + " " + s.facts(t, rec, factOpts{Made: true, Boxed: box != "", Chips: true, From: s.from(r, t, rec)}) + `</div>`)
}

// howMany says how many there are under a listing's title, and how many
// of them are done when the type keeps that.
func howMany(t *schema.Type, recs []*store.Record) template.HTML {
	n := len(recs)
	what := plural(t.Name)
	if n == 1 {
		what = schema.Words(t.Name)
	}
	text := fmt.Sprintf("%d %s", n, what)
	if f := doneField(t); f != nil {
		done := 0
		for _, rec := range recs {
			if on, _ := rec.Fields[f.Name].(bool); on {
				done++
			}
		}
		if done > 0 {
			text += fmt.Sprintf(" · %d %s", done, strings.ToLower(label(f.Name)))
		}
	}
	return template.HTML(`<p class="sw-lede">` + template.HTMLEscapeString(text) + `</p>`)
}

// factOpts says how the facts are shown: Made adds when the record was
// made; Boxed leaves out the done chip because a box already shows it;
// Chips draws the day and what it belongs to as chips rather than words,
// and words are short, the way a row says them.
type factOpts struct {
	Made, Boxed, Chips bool
	// From is who wrote the record's words, with Made; see from.
	From string
}

// facts is what a record says at a glance (glance.go), as chips under its
// title or short words at a row's right: the same on its own page as in
// its list. With no day to say, a row says when the record last changed;
// the record's own page says when it was made.
func (s *Server) facts(t *schema.Type, rec *store.Record, o factOpts) string {
	now := time.Now()
	facts := s.glance(t, rec, now)
	day := false
	for _, f := range facts {
		day = day || f.Kind == "day"
	}
	boxed := ""
	if o.Boxed {
		boxed = s.boxed(t, rec)
	}
	out := s.glanceHTML(facts, o.Chips, boxed)
	if !day && !o.Made && !hasDate(t) {
		out = strings.TrimSpace(out + ` <span class="sw-muted">` + when.Relative(rec.UpdatedAt.UTC().Format(time.RFC3339), now) + `</span>`)
	}
	if o.Made {
		out = strings.TrimSpace(out + " " + whenMade(t, rec, o.From))
	}
	return out
}

// whenMade says when a record was made, as a person reads a time, and where
// its words came from when that was not the owner, in one quiet line under
// its fields. No field-name label prefix: just the relative timestamp text so
// dates read naturally without "Added" or "Started" before them. Rendered
// output: <span class="sw-detail__when sw-muted sw-small">4 days ago at 1:31pm</span>
func whenMade(t *schema.Type, rec *store.Record, from string) string {
	made := when.Relative(rec.CreatedAt.UTC().Format(time.RFC3339), time.Now())
	line := made
	if from != "" {
		line += " · From: " + template.HTMLEscapeString(from)
	}
	return `<span class="sw-detail__when sw-muted sw-small">` + line + `</span>`
}

// from is who wrote a record's words, said once on its page, when that
// was someone other than the owner and their assistant: an import, another
// person, an agent, an action. A file's or a device's page already says
// what it is. A reader from the internet is not told anyone's name.
func (s *Server) from(r *http.Request, t *schema.Type, rec *store.Record) string {
	if t.Name == FileType || t.Name == "device" {
		return ""
	}
	if w := s.app.Chat.For(chat.VisitorOf(r.Context())).Writers().Of(t.Name, rec); w.Outside {
		return w.Words
	}
	return ""
}

// dotOf is the colour a list wears everywhere it appears: the sidebar,
// a page title, the crumbs, a block drawn from it, a search result. The
// person's own types take the six list colours in order; an internal
// type has none.
// The colours go to the lists a person sees, in order, so the few on show
// each have their own; a type with nothing in it yet counts only for itself.
func (s *Server) dotOf(typeName string) int {
	n := 0
	for _, t := range s.app.Types.Types {
		if t.Internal || (t.Name != typeName && !s.listed(t)) {
			continue
		}
		n++
		if t.Name == typeName {
			return (n-1)%6 + 1
		}
	}
	return 0
}

// hasDate says whether a type has a day of its own, such as a task's due.
func hasDate(t *schema.Type) bool {
	for _, f := range t.Shown() {
		if f.Type == "datetime" {
			return true
		}
	}
	return false
}
