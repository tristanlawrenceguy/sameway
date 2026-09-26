package server

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The few words a page says about a record beside its title: its box, its
// state, the day that matters to it, what it belongs to, when it was made.

// lede is the line under a record's title: its box when it has one, its
// facts as chips, then when it was made. On detail pages the mark carries
// aria-label so screen readers hear the state from the checkbox alone; the
// lede text starts directly with badges — no leading state word at all.
func (s *Server) lede(t *schema.Type, rec *store.Record) template.HTML {
	box := ""
	if props, ok := markOf(t, rec); ok {
		// quiet=true + ariaLabel means the mark renders nothing visible or
		// visually-hidden: only the checkbox with its aria-label carries it.
		props["quiet"] = true
		box = string(s.component("mark", props))
	}
	return template.HTML(`<p class="sw-lede">` + box + s.facts(t, rec, factOpts{Made: true, Boxed: box != "", Chips: true}) + `</p>`)
}

// howMany says how many there are under a listing's title, and how many
// of them are done when the type keeps that.
func howMany(t *schema.Type, recs []*store.Record) template.HTML {
	n := len(recs)
	what := plural(t.Name)
	if n == 1 {
		what = t.Name
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
type factOpts struct{ Made, Boxed, Chips bool }

// facts is what a person wants to know about a record at a glance: done,
// its state, the day that matters, what it belongs to. A day that has
// passed on something not done is amber, with the word "was". When there
// is nothing of the kind, when it last changed.
func (s *Server) facts(t *schema.Type, rec *store.Record, o factOpts) string {
	var parts []string
	done := false
	if f := doneField(t); f != nil {
		if v, _ := rec.Fields[f.Name].(bool); v {
			done = true
			if !o.Boxed {
				parts = append(parts, string(s.component("badge", map[string]any{"label": capitalize(label(f.Name)), "tone": "success"})))
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
					parts = append(parts, string(s.component("badge", map[string]any{"label": capitalize(label(f.Name)), "tone": "neutral"})))
				}
			}
		}
	}
	for _, f := range t.Shown() {
		if f.Type == "enum" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				// "High", heard in a row of facts, says nothing without the
				// field it is: the field is read after it, not shown.
				parts = append(parts, string(s.component("badge", map[string]any{"label": f.ValueLabel(v), "tone": "info", "context": strings.ToLower(fieldLabel(f))})))
			}
			break
		}
	}
	// When it last changed is said only for things with no day of their
	// own, such as a note; a task with no due date has nothing to add.
	if d := s.dayFact(t, rec, done, o.Chips); d != "" {
		parts = append(parts, d)
	} else if !o.Made && !hasDate(t) {
		parts = append(parts, `<span class="sw-muted">Updated `+when.Short(rec.UpdatedAt.UTC().Format(time.RFC3339), time.Now())+`</span>`)
	}
	for _, f := range t.Shown() {
		if f.Type == "ref" {
			if id, ok := rec.Fields[f.Name].(string); ok && id != "" {
				if title := s.refTitle(f, id); title != "" {
					if o.Chips {
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
		parts = append(parts, whenMade(rec))
	}
	return strings.Join(parts, " ")
}

// dayFact is the record's first date: short at the right of a row, in full
// as a chip under a title; amber with "was" when it has passed undone.
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
			// human tone, which says who did a thing.
			text, tone := label(f.Name)+" "+when.Text(v), "info"
			if past {
				text, tone = "Was "+strings.ToLower(label(f.Name))+" "+when.Text(v), "warning"
			}
			return string(s.component("badge", map[string]any{"label": text, "tone": tone}))
		}
		short := when.Short(v, now)
		class := "sw-when"
		switch {
		case past:
			short, class = "Was "+strings.ToLower(label(f.Name))+" "+short, "sw-when sw-when--past"
		case strings.HasPrefix(short, "Today"):
			class = "sw-when sw-when--today"
		}
		return `<span class="` + class + `">` + template.HTMLEscapeString(short) + `</span>`
	}
	return ""
}

// whenMade says when a record was made and last changed, as a person reads
// a time, in one quiet line under its fields.
func whenMade(rec *store.Record) string {
	made := when.Text(rec.CreatedAt.UTC().Format(time.RFC3339))
	changed := when.Text(rec.UpdatedAt.UTC().Format(time.RFC3339))
	if changed == made {
		return `<span class="sw-detail__when sw-muted sw-small">Created ` + made + `</span>`
	}
	return `<span class="sw-detail__when sw-muted sw-small">Created ` + made + ` · Updated ` + changed + `</span>`
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
