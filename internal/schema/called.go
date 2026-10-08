package schema

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Called is what a record of this type is called, from its fields alone:
// its title field, or else the first thing it says (a choice by its
// label), or else its kind and id. records.Name, which every surface names a
// record by, adds what needs the store (an entry by its habit, a change by
// its sentence); a package below chat, an export or a query matching a
// ref by its title, calls this, so a record with no title is called the
// same there as on its page.
func (t *Type) Called(id string, fields map[string]any) string {
	if t.Title != "" {
		if s, ok := fields[t.Title].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	for _, f := range t.Shown() {
		if f.Type != "string" && f.Type != "text" && f.Type != "enum" {
			continue
		}
		if s, ok := fields[f.Name].(string); ok && strings.TrimSpace(s) != "" {
			if f.Type == "enum" {
				s = f.ValueLabel(s)
			}
			return trim.Title(s)
		}
	}
	return Words(t.Name) + " " + id
}
