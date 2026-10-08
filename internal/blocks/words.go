package blocks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Words a block and a page say alike: a stored value as text, a list as
// said, why a type is not there.

// Display renders a stored value as the text a form or page shows.
func Display(f schema.Field, v any) string {
	if v == nil {
		return ""
	}
	switch f.Type {
	case "list":
		if s, ok := v.(string); ok {
			return s // a value the person just typed, coming back after an error
		}
		items, _ := v.([]any)
		parts := make([]string, 0, len(items))
		for _, it := range items {
			parts = append(parts, fmt.Sprint(it))
		}
		if f.Multiline {
			return strings.Join(parts, "\n")
		}
		return strings.Join(parts, ", ")
	case "json":
		if s, ok := v.(string); ok {
			return s
		}
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	case "bool":
		if b, _ := v.(bool); b {
			return "yes"
		}
		return "no"
	case "datetime":
		return when.Text(fmt.Sprint(v))
	case "repeat":
		if said := when.RepeatText(fmt.Sprint(v)); said != "" {
			return Capitalize(said)
		}
		return ""
	case "int":
		if n, ok := v.(int64); ok && f.Name == "size" {
			return SizeWords(n)
		}
		if n, ok := v.(int); ok && f.Name == "size" {
			return SizeWords(int64(n))
		}
		return fmt.Sprint(v)
	case "enum":
		return f.ValueLabel(fmt.Sprint(v))
	}
	return fmt.Sprint(v)
}

// SizeWords is a file's size as a person reads it: 11 MB, 480 KB.
func SizeWords(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// Capitalize puts a phrase at the start of a sentence.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + s[1:]
}

// Strs reads a list of strings out of props, whatever JSON made of it.
func Strs(v any) []string {
	var out []string
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		for _, it := range x {
			if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
	case string:
		if strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}

// str reads a string, falling back when there is none.
func str(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

// AndList is names as said: tasks, reminders and entries.
func AndList(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// NoType says a type is not there, and what is: the one likely meant,
// when the name is near one, then every one.
func (w *Workspace) NoType(name string) string {
	names := w.Store.Types().Names()
	out := "there is no content type " + name
	if near := render.Nearest(name, names); near != "" {
		out += " (did you mean " + near + "?)"
	}
	return out + "; the workspace has " + strings.Join(names, ", ")
}

// FieldsOfKind is the names of a type's fields of the kinds given, or
// every field when none is given.
func FieldsOfKind(t *schema.Type, kinds ...string) []string {
	var out []string
	for _, f := range t.Shown() {
		if len(kinds) == 0 {
			out = append(out, f.Name)
			continue
		}
		for _, k := range kinds {
			if f.Type == k {
				out = append(out, f.Name)
			}
		}
	}
	return out
}

// NothingYet is a block that shows nothing, said so a model cannot read
// it as done: which records it waits for (has: a field they need set),
// and why it is not refused. Records may be added later (a list of books
// to read, before the first book), so it is written; if some should show
// now, the conditions are wrong, and the writer is the one to know.
func NothingYet(typeName string, where []string, has string) string {
	one := schema.Words(typeName)
	out := "nothing yet: there are no " + schema.Plural(typeName)
	switch {
	case has != "" && len(where) > 0:
		out = "nothing yet: no " + one + " has a " + has + " and matches " + strings.Join(where, " and ")
	case has != "":
		out = "nothing yet: no " + one + " has a " + has
	case len(where) > 0:
		out = "nothing yet: no " + one + " matches " + strings.Join(where, " and ")
	}
	return out + " (it fills in as records are added; if some should show now, change the conditions)"
}

// Listed says whether a list belongs in the sidebar: one with something
// in it, or one the person made themselves, which they will want to see
// even before its first record. An empty list the system provides, such
// as files in a workspace with no files, is not in the way. ui.lists: all
// shows every one.
func (w *Workspace) Listed(t *schema.Type) bool {
	if t.Hidden {
		return false
	}
	if w.listsAll() || !t.Provided {
		return true
	}
	n, err := w.Store.Count(t.Name)
	return err != nil || n > 0
}
