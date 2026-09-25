package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// display renders a stored value as the text a form or page shows.
func display(f schema.Field, v any) string {
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
	case "enum":
		return f.ValueLabel(fmt.Sprint(v))
	}
	return fmt.Sprint(v)
}

// titleOf names a record: its title field, else the first string field
// with something in it, else its type and id. A blank title used to fall
// straight to the id, so a list of activities read as a column of "said".
func titleOf(t *schema.Type, rec *store.Record) string {
	if t.Title != "" {
		if s, ok := rec.Fields[t.Title].(string); ok && s != "" {
			return s
		}
	}
	for _, f := range t.Shown() {
		if f.Type != "string" && f.Type != "text" && f.Type != "enum" {
			continue
		}
		if s, ok := rec.Fields[f.Name].(string); ok && strings.TrimSpace(s) != "" {
			return truncateTitle(s)
		}
	}
	return t.Name + " " + rec.ID
}

// fieldLabel is what a person calls a field: the schema's label, else its
// name made readable.
func fieldLabel(f schema.Field) string {
	if f.Label != "" {
		return f.Label
	}
	return label(f.Name)
}
