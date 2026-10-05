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
	return template.HTML(`<div class="sw-lede">` + box + " " + s.facts(t, rec, factOpts{Made: true, Boxed: box != "", Chips: true, Detail: true, From: s.from(r, t, rec)}) + `</div>`)
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
	Made, Boxed, Chips, Row, Detail bool
	// From is who wrote the record's words, with Made; see from.
	From string
}

// facts is what a person wants to know about a record at a glance: done,
// its state, the day that matters, what it belongs to. A day that has
// passed on something not done is amber, with the word "was". When there
// is nothing of the kind, when it last changed.
func (s *Server) facts(t *schema.Type, rec *store.Record, o factOpts) string {
	var parts []string
	// An entry is a thing done: its day is when it happened, never "was".
	done := t.Name == EntryType
	if f := doneField(t); f != nil {
		if v, _ := rec.Fields[f.Name].(bool); v {
			done = true
			if !o.Boxed {
				parts = append(parts, string(s.component("badge", map[string]any{"label": fieldLabel(*f), "tone": "success"})))
			}
		}
	}
	// A setting that is on, such as pinned or show, is said as a badge — but
	// not when the mark checkbox already carries it (Boxed=true), since that
	// would repeat the same fact twice.
	if !o.Boxed || doneField(t) != nil {
		for _, f := range t.Shown() {
			if f.Type == "bool" && (doneField(t) == nil || f.Name != doneField(t).Name) {
				if on, _ := rec.Fields[f.Name].(bool); on {
					parts = append(parts, string(s.component("badge", map[string]any{"label": fieldLabel(f), "tone": "neutral"})))
				}
			}
		}
	}
	// Show enum badges only when the caller is not a list row for note/project/file/reminder,
	// and not an action detail (actions hide their kind everywhere). Also skip
	// note/project/file/task/habit/reminder on detail pages — the status badge in meta text repeats what
	// the definition list below already says.
	skipEnum := o.Row && (t.Name == "note" || t.Name == "project" || t.Name == "file" || t.Name == ReminderType)
	detailSkip := o.Detail && (t.Name == "note" || t.Name == "project" || t.Name == "file" || t.Name == "task" || t.Name == HabitType || t.Name == ReminderType)
	if !skipEnum && !detailSkip && t.Name != "action" {
		for _, f := range t.Shown() {
			if f.Type == "enum" {
				// A task's status says only Doing: To do and Done are its tick.
				if v, ok := rec.Fields[f.Name].(string); ok && v != "" && (f.Name != t.Stage() || t.SaysMoreThanTick(v)) {
					props := map[string]any{"label": f.ValueLabel(v), "tone": "info"}
					if f.Labels[v] == "" {
						props["context"] = strings.ToLower(fieldLabel(f))
					}
					parts = append(parts, string(s.component("badge", props)))
				}
				break
			}
		}
	}

	// The record's first date: short at the right of a row, in full as a chip.
	if d := s.dayFact(t, rec, done, o.Chips && t.Name != EntryType); d != "" {
		parts = append(parts, d)
	} else if !o.Made && !hasDate(t) {
		parts = append(parts, `<span class="sw-muted">`+when.Relative(rec.UpdatedAt.UTC().Format(time.RFC3339), time.Now())+`</span>`)
	}

	// An entry's row is already titled by its habit; saying it again under the title is the same words twice.
	for _, f := range t.Shown() {
		if f.Type == "ref" && (o.Chips || t.Name != EntryType) {
			if id, ok := rec.Fields[f.Name].(string); ok && id != "" {
				if title := s.refTitle(f, id); title != "" {
					// Someone it is for: their name, with their colour.
					if f.To == chat.PersonType {
						parts = append(parts, s.personChip(fieldLabel(f), id, title))
					} else if o.Chips {
						parts = append(parts, string(s.component("badge", map[string]any{"label": title, "tone": "neutral"})))
					} else {
						parts = append(parts, `<span class="sw-row__note">`+template.HTMLEscapeString(title)+`</span>`)
					}
				}
			}
			break
		}
	}

	if o.Made {
		parts = append(parts, whenMade(t, rec, o.From))
	}
	return strings.Join(parts, " ")
}

// dayFact is the record's first date: short at the right of a row, in full
// as a chip under a title; amber with "was" when it has passed undone.
// When no custom Label exists on the field, only the date text appears — no
// raw schema column name prefix (see TestTaskDetailLedeNoRawDueLabel).
func (s *Server) dayFact(t *schema.Type, rec *store.Record, done, chip bool) string {
	for _, f := range t.Shown() {
		if f.Type != "datetime" {
			continue
		}
		v, ok := rec.Fields[f.Name].(string)
		if !ok || v == "" {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, v)
		now := time.Now()
		dayOnly := strings.HasSuffix(v, "T00:00:00Z")
		past := !done && (!dayOnly && ts.Before(now) || dayOnly && ts.AddDate(0, 0, 1).Before(now))

		if chip {
			// A day is a fact, not something a person did: info, not the
			// human tone, which says who did a thing. Use relative text so
			// dates read in natural language ("Tomorrow at 10am") rather than
			// machine format ("Mon 5 Oct 2026, 14:00").
			text := when.Relative(v, now)
			tone := "info"
			if f.Label != "" {
				text = f.Label + " " + text
			}
			if past && f.Label != "" {
				text, tone = "Was "+strings.ToLower(f.Label)+" "+when.Relative(v, now), "warning"
			} else if past {
				text, tone = when.Relative(v, now), "warning"
			}
			return string(s.component("badge", map[string]any{"label": text, "tone": tone}))
		}
		short := when.Relative(v, now)
		class := "sw-when"
		switch {
		case past && f.Label != "":
			short, class = "Was "+strings.ToLower(f.Label)+" "+short, "sw-when sw-when--past"
		case strings.HasPrefix(short, "Today"):
			class = "sw-when sw-when--today"
		}
		return `<span class="` + class + `">` + template.HTMLEscapeString(short) + `</span>`
	}
	return ""
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
