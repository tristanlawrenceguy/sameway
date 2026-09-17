package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The shape of the content is the person's too: a new property on their
// tasks, or a new kind of thing altogether, is one ask away. These tools
// change the workspace's schema for everyone, so the reply says so.

// fieldDef is what the model gives for one field, in either tool.
type fieldDef struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Values      []string `json:"values"`
	To          string   `json:"to"`
	Required    bool     `json:"required"`
	Default     any      `json:"default"`
}

func (d fieldDef) field() schema.Field {
	return schema.Field{Name: strings.TrimSpace(d.Name), Type: d.Kind, Description: d.Description, Values: d.Values, To: d.To, Required: d.Required, Default: d.Default}
}

var fieldProps = map[string]any{
	"name":        map[string]any{"type": "string", "description": "Lowercase letters, digits and underscores, such as due or priority."},
	"kind":        map[string]any{"type": "string", "enum": schema.FieldTypes, "description": "string is a line of text, text a paragraph, markdown structured text, datetime a day or a moment, enum one of values, list several strings, ref another record's id (say which type in to), bool yes or no."},
	"description": map[string]any{"type": "string", "description": "What the field is for, in a few words: shown to people and to you."},
	"values":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "For enum: the choices."},
	"to":          map[string]any{"type": "string", "description": "For ref: the content type it points at."},
	"required":    map[string]any{"type": "boolean"},
	"default":     map[string]any{"description": "What a record has when nothing was given."},
}

func shapeTools() []llm.Tool {
	obj := func(props map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	return []llm.Tool{
		{Name: "add_field", Description: "Add a property to a content type, for everyone: a due date on notes, a priority on tasks. The type's schema file and its table change at once, and every record has the field from then on. Adding is safe; nothing existing changes.",
			Schema: obj(map[string]any{
				"type":        map[string]any{"type": "string", "description": "The content type to add the field to."},
				"name":        fieldProps["name"],
				"kind":        fieldProps["kind"],
				"description": fieldProps["description"],
				"values":      fieldProps["values"],
				"to":          fieldProps["to"],
				"required":    fieldProps["required"],
				"default":     fieldProps["default"],
			}, "type", "name", "kind")},
		{Name: "add_type", Description: "Make a new content type, for everyone: a kind of thing the person keeps, such as habit, contact or recipe, with its own page at /t/<name>, its own records and its own fields. Give the title field first.",
			Schema: obj(map[string]any{
				"name":        map[string]any{"type": "string", "description": "Singular, lowercase, such as contact."},
				"description": map[string]any{"type": "string", "description": "One sentence: what one of these is."},
				"properties":  map[string]any{"type": "array", "items": obj(fieldProps, "name", "kind"), "description": "The fields, the title first (a string)."},
			}, "name", "properties")},
	}
}

func (s *Service) addField(typeName string, d fieldDef) toolResult {
	if s.AddField == nil {
		return fail("this workspace cannot change its schema from here")
	}
	t, err := s.AddField(strings.ToLower(strings.TrimSpace(typeName)), d.field())
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("added %s (%s) to %s; every %s has it now, and its page at /t/%s shows it", d.Name, d.Kind, t.Name, t.Name, t.Name),
		change: &Change{Action: "added", Component: "field", Detail: d.Name + " on " + t.Name, Href: "/t/" + t.Name},
	}
}

func (s *Service) addType(name, description string, defs []fieldDef) toolResult {
	if s.AddType == nil {
		return fail("this workspace cannot change its schema from here")
	}
	t := &schema.Type{Name: strings.ToLower(strings.TrimSpace(name)), Description: description}
	for _, d := range defs {
		t.Fields = append(t.Fields, d.field())
	}
	made, err := s.AddType(t)
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("made the content type %s with fields %s; its records live at /t/%s, and create_record makes one", made.Name, fieldNames(made), made.Name),
		change: &Change{Action: "added", Component: "type", Detail: made.Name, Href: "/t/" + made.Name},
	}
}

func fieldNames(t *schema.Type) string {
	var names []string
	for _, f := range t.Fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}
