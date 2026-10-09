package server

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What a record says at a glance is worked out in internal/records
// (glance.go there), from the schema and the store; here it is shown: as
// chips under a record's title, or short words at a row's right.

// glanceHTML is the facts as a page shows them: chips under a title, or
// short words at a row's right. boxed is the field a box beside them
// already shows (done, pinned), which is not said again.
func (s *Server) glanceHTML(facts []records.Fact, chips bool, boxed string) string {
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
		case (f.Kind == "ref" || f.Kind == "count") && !chips:
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
	// belongs to is, being a way there the chip is not. What is in it has
	// no field of its own, so it is not counted for this.
	for _, f := range records.Glance(s.app.Store, t, rec, s.now(), s.h24(), records.Counts{}) {
		if f.Kind != "ref" && f.Kind != "person" {
			out[f.Field] = true
		}
	}
	// A state not worth saying (To do, Draft) is said nowhere at rest; one
	// worth saying is said in the fields, where it is changed.
	for _, f := range t.Shown() {
		if worth, state := records.StateWorth(t, f, fmt.Sprint(rec.Fields[f.Name])); state && !worth {
			out[f.Name] = true
		}
	}
	return out
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

// timeHTML puts what a day says in a <time> holding its value, with the
// date in full as its title when the words leave it out (Today). A screen
// reader reads the words, which say it on their own; the value is for a
// machine, the title for a pointer.
func timeHTML(class, value, full string, inner template.HTML) string {
	b := "<time"
	if class != "" {
		b += ` class="` + class + `"`
	}
	if value != "" {
		b += ` datetime="` + template.HTMLEscapeString(value) + `"`
	}
	if full != "" {
		b += ` title="` + template.HTMLEscapeString(full) + `"`
	}
	return b + ">" + string(inner) + "</time>"
}
