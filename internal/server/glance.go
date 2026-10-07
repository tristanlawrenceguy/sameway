package server

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What a record says at a glance is worked out once, here, from its
// schema, and every surface shows the same facts its own way: the line
// under a record's title as chips, a list's row as short words, a block's
// list as plain text. Three functions once said it three ways, each
// patched by the type it was caught on, and the same record read "Due Fri
// 9 Oct 2026, 14:00" on the canvas and "In 4 days at 2:00pm" on its list.

// glanceFact is one thing a record says at a glance.
type glanceFact struct {
	Kind   string // done, state, flag, day, person, ref
	Field  string // the field it says, which the record's page need not list again
	Text   string // as a person reads it
	Tone   string // the badge's tone
	Person string // a person's id, for their colour
	Label  string // a person's field's label: For
	Short  string // a day as a row says it, a class to go with it
	Class  string
	When   string // a day's value, for its <time datetime>
	Full   string // a day in full, when its words leave the date out
}

// glance is what a record says at a glance, by the schema's rules, not by
// its type's name:
//   - its done tick, when ticked;
//   - its stage when it says more than the tick (Doing), or its status or
//     state (Draft, Pending); a kind, a method, a cadence is the record's
//     own page's to say;
//   - a setting that is on, by its label (Pinned, On the canvas);
//   - its first day, in words, named by its field (Due Fri 9 Oct,
//     Starts tomorrow at 2pm), and "Overdue" in words and amber when it
//     has passed on a record that can be done and is not; a day that has
//     passed on anything else (a meeting, an entry) is only past;
//   - what it belongs to or who it is for, unless that is its title; a
//     file it points at is an attachment, not what it belongs to.
//
// The order is always this one, so the same fact sits in the same place
// on every row (design/foundations/glance.md).
func (s *Server) glance(t *schema.Type, rec *store.Record, now time.Time) []glanceFact {
	var out []glanceFact
	done := false
	if f := doneField(t); f != nil {
		if on, _ := rec.Fields[f.Name].(bool); on {
			done = true
			out = append(out, glanceFact{Kind: "done", Field: f.Name, Text: fieldLabel(*f), Tone: "success"})
		}
	}
	for _, f := range t.Shown() {
		v, _ := rec.Fields[f.Name].(string)
		if f.Type != "enum" || v == "" {
			continue
		}
		if worth, said := stateWorth(t, f, v); said && worth {
			out = append(out, glanceFact{Kind: "state", Field: f.Name, Text: f.ValueLabel(v), Tone: "info"})
		}
	}
	for _, f := range t.Shown() {
		if f.Type == "bool" && (doneField(t) == nil || f.Name != doneField(t).Name) {
			if on, _ := rec.Fields[f.Name].(bool); on {
				out = append(out, glanceFact{Kind: "flag", Field: f.Name, Text: fieldLabel(f), Tone: "neutral"})
			}
		}
	}
	if d, ok := dayGlance(t, rec, done, now); ok {
		out = append(out, d)
	}
	title := strings.TrimSpace(chat.Name(s.app.Store, t, rec))
	for _, f := range t.Shown() {
		id, _ := rec.Fields[f.Name].(string)
		if f.Type != "ref" || id == "" || f.To == FileType {
			continue
		}
		if name := s.RefTitle(f, id); name != "" && name != title {
			if f.To == chat.PersonType {
				out = append(out, glanceFact{Kind: "person", Field: f.Name, Text: fieldLabel(f) + " " + name, Person: id, Label: fieldLabel(f)})
			} else {
				out = append(out, glanceFact{Kind: "ref", Field: f.Name, Text: name, Tone: "neutral"})
			}
		}
		break
	}
	return out
}

// glanceText is the facts as plain words, for a block's list.
func (s *Server) glanceText(t *schema.Type, rec *store.Record) string {
	var words []string
	for _, f := range s.glance(t, rec, time.Now()) {
		words = append(words, f.Text)
	}
	return strings.Join(words, " · ")
}

// glanceHTML is the facts as a page shows them: chips under a title, or
// short words at a row's right. boxed is the field a box beside them
// already shows (done, pinned), which is not said again.
func (s *Server) glanceHTML(facts []glanceFact, chips bool, boxed string) string {
	var parts []string
	for _, f := range facts {
		switch {
		case f.Field != "" && f.Field == boxed:
		case f.Kind == "person":
			parts = append(parts, s.personChip(f.Label, f.Person, strings.TrimPrefix(f.Text, f.Label+" ")))
		case f.Kind == "day" && !chips:
			parts = append(parts, timeHTML(f.Class, f.When, f.Full, template.HTML(template.HTMLEscapeString(f.Short))))
		case f.Kind == "day":
			parts = append(parts, timeHTML("", f.When, f.Full, s.component("badge", map[string]any{"label": f.Text, "tone": f.Tone})))
		case f.Kind == "ref" && !chips:
			parts = append(parts, `<span class="sw-row__note">`+template.HTMLEscapeString(f.Text)+`</span>`)
		default:
			parts = append(parts, string(s.component("badge", map[string]any{"label": f.Text, "tone": f.Tone})))
		}
	}
	return strings.Join(parts, " ")
}

// headFields are the fields a record's page already says above its list
// of fields: its title as the heading, its box, and what its glance says.
// Taken from the glance itself, so the two cannot drift apart: they did,
// and a task's status was said in neither.
func (s *Server) headFields(t *schema.Type, rec *store.Record) map[string]bool {
	out := map[string]bool{}
	if t.Title != "" {
		out[t.Title] = true
	}
	// A file's path is an id that means nothing to a person; its page
	// says where the original is with Open the original.
	if t.Name == FileType {
		out["path"] = true
	}
	out[s.boxed(t, rec)] = true
	// What the line above says is not said again; a link to what it
	// belongs to is, being a way there the chip is not.
	for _, f := range s.glance(t, rec, time.Now()) {
		if f.Kind != "ref" && f.Kind != "person" {
			out[f.Field] = true
		}
	}
	// A state not worth saying (To do, Draft) is said nowhere at rest; one
	// worth saying is said in the fields, where it is changed.
	for _, f := range t.Shown() {
		if worth, state := stateWorth(t, f, fmt.Sprint(rec.Fields[f.Name])); state && !worth {
			out[f.Name] = true
		}
	}
	return out
}

// stateWorth says whether a field is a record's state (its stage, or a
// status or state) and whether its value is worth saying: a stage when it
// says more than the tick (Doing; To do and Done are the tick), a status
// or state when it is not where every record rests (Published, not
// Draft), or when it waits on someone (Pending).
func stateWorth(t *schema.Type, f schema.Field, v string) (worth, state bool) {
	if f.Type != "enum" {
		return false, false
	}
	if f.Name == t.Stage() {
		return t.SaysMoreThanTick(v), true
	}
	if f.Name != "status" && f.Name != "state" {
		return false, false
	}
	rest, _ := f.Default.(string)
	if rest == "" && len(f.Values) > 0 {
		rest = f.Values[0]
	}
	return v != rest || v == "pending", true
}

// boxed is the field a record's own page shows as a box (done, pinned),
// or "".
func (s *Server) boxed(t *schema.Type, rec *store.Record) string {
	if props, ok := s.markOf(t, rec); ok {
		f, _ := props["field"].(string)
		return f
	}
	return ""
}
