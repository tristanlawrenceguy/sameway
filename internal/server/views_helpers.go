package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
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
	case "repeat":
		if said := when.RepeatText(fmt.Sprint(v)); said != "" {
			return capitalize(said)
		}
		return ""
	case "enum":
		return f.ValueLabel(fmt.Sprint(v))
	}
	return fmt.Sprint(v)
}

// fieldLabel is what a person calls a field: the schema's label, else its
// name made readable.
func fieldLabel(f schema.Field) string {
	if f.Label != "" {
		return f.Label
	}
	return label(f.Name)
}
