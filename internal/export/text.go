package export

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A record as text, for a reader rather than a spreadsheet: a published
// record over MCP, a record taken out as Markdown, a value the assistant
// says back. Each once wrote its own: one gave raw choice values and
// stored timestamps under raw field names, one a choice's label and an
// ISO day, one whatever was kept. They now say a field by its name
// (schema Display), a choice by its label, a day as a person reads it
// (when.Full) and a ref by its title.

// Said is one field's value as a reader reads it: Value, but a day or a
// moment in full words, Monday 5 October 2026 at 2pm, not 2026-10-05 14:00,
// on the 24-hour clock when h24 is the reader's (when.TwentyFour).
func Said(f schema.Field, v any, titles Titles, h24 bool) string {
	if f.Type == "datetime" && v != nil {
		return when.Full(fmt.Sprint(v), h24)
	}
	return Value(f, v, titles)
}

// Fact is one field of a record as words.
type Fact struct {
	Field schema.Field
	Name  string // what the field is called
	Value string
}

// Text is a record's shown fields as words, but its title, which the
// reader is given as its name: facts, each field with something in it
// and not a plain no, and body, its writing (text and markdown fields),
// whole, in field order. With titles nil a ref is left out, not given as
// an id that means nothing to a reader.
func Text(t *schema.Type, fields map[string]any, titles Titles, h24 bool) (facts []Fact, body []string) {
	for _, f := range t.Shown() {
		v := fields[f.Name]
		if f.Name == t.Title || v == nil || v == "" || (titles == nil && (f.Type == "ref" || f.RefList())) {
			continue
		}
		if s, ok := v.(string); ok && (f.Type == "text" || f.Type == "markdown") {
			if s = strings.TrimSpace(s); s != "" {
				body = append(body, s)
			}
			continue
		}
		if said := Said(f, v, titles, h24); said != "" && said != "no" {
			facts = append(facts, Fact{Field: f, Name: f.Display(), Value: said})
		}
	}
	return facts, body
}
