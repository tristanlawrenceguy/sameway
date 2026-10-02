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
	"kind":        map[string]any{"type": "string", "enum": schema.FieldTypes, "description": "string is a line of text, text a paragraph, markdown structured text, datetime a day or a moment, repeat how often it happens again (every Tuesday), enum one of values, list several strings, ref another record's id (say which type in to), bool yes or no."},
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
		{Name: "add_field", Description: "Add a property to a content type, for everyone: a due date on notes, a priority on tasks. The type's schema file and its table change at once, and every record has the field from then on: the ones already there read as its default (nothing, when it has none), and the answer says how many there are and what they got. Adding is safe; nothing else they hold changes.",
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
	// A model asked to organise writing reached for a field of its own
	// ("part of") two times in three; the tool that does it is said here.
	if writingField[strings.NewReplacer(" ", "_", "-", "_").Replace(strings.ToLower(d.Name))] {
		return fail("pieces of writing in parts are organised with organise_writing, not a field of your own: it adds part_of, parts and material_for itself and sets the parts in order and the material in one change. Call organise_writing with piece (the whole), parts (their ids in order) and material (guidelines, research and the like). add_field is for other properties.")
	}
	t, err := s.AddField(strings.ToLower(strings.TrimSpace(typeName)), d.field())
	if err != nil {
		return fail("%v", err)
	}
	gets := ""
	if said := FieldGets(s.Store, t, d.Name); said != "" {
		gets = " (" + said + ")"
	}
	return toolResult{
		text:   fmt.Sprintf("added %s (%s) to %s; every %s has it now%s, and its page at /t/%s shows it", d.Name, d.Kind, t.Name, t.Name, gets, t.Name),
		change: &Change{Action: "added", Component: "field", Detail: d.Name + " on " + schema.Words(t.Name), Href: "/t/" + t.Name},
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
		change: &Change{Action: "added", Component: "type", Detail: schema.Words(made.Name), Href: "/t/" + made.Name},
	}
}

func fieldNames(t *schema.Type) string {
	var names []string
	for _, f := range t.Fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}

// writingField is a field name that means a piece and its parts, which
// organise_writing keeps.
var writingField = map[string]bool{"part_of": true, "partof": true, "parts": true, "parent": true, "chapter_of": true,
	"section_of": true, "belongs_to": true, "chapters": true, "sections": true, "material_for": true, "material": true}
