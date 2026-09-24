package server

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person edits the whole record where it is. The page at rest says the
// least it can: the title in the heading, a few chips, the words, and no
// empty fields, so there is nothing on it for an editor to find the rest
// by. Before this, Edit offered only what the page happened to show: the
// title, the chips and every empty field could be changed by asking the
// assistant and no other way, and an empty record had no Edit at all.
// Now the record says what can be edited, once, in a template the page
// does not show; Edit builds the form from it, every field of the type in
// its order, as the controls the inline editor already makes.

// editFields is the template of what a person can edit on a record: every
// field of its type but those the system keeps, marked as the inline
// editor reads them. Empty for a type the system keeps.
func (s *Server) editFields(t *schema.Type, rec *store.Record) string {
	if t.Internal {
		return ""
	}
	var b strings.Builder
	for _, f := range t.Fields {
		if f.ReadOnly || f.Type == "json" {
			continue
		}
		if el := s.editField(f, rec.Fields[f.Name]); el != "" {
			b.WriteString(el)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return `<template data-edit-fields>` + b.String() + `</template>`
}

// editField is one field as the inline editor reads it: the name, what is
// in it now as written, and for a field with set values, the values.
func (s *Server) editField(f schema.Field, v any) string {
	val := display(f, v)
	esc := template.HTMLEscapeString
	name, lab := esc(f.Name), ` data-label="`+esc(label(f.Name))+`"`
	if f.Label != "" {
		lab = ` data-label="` + esc(f.Label) + `"`
	}
	switch f.Type {
	case "markdown":
		return fmt.Sprintf(`<div class="sw-prose" data-prop="%s"%s data-source="%s" data-prose-level="2">%s</div>`, name, lab, esc(val), prose.Render(val, 2))
	case "text", "string", "list":
		if f.Type != "text" && !f.Multiline {
			break
		}
		return fmt.Sprintf(`<div data-prop="%s"%s data-source="%s">%s</div>`, name, lab, esc(val), esc(val))
	case "datetime":
		raw := ""
		if v != nil {
			raw = fmt.Sprint(v)
		}
		return fmt.Sprintf(`<span data-prop="%s"%s data-kind="datetime" data-source="%s">%s</span>`, name, lab, esc(raw), esc(val))
	case "int", "float":
		return fmt.Sprintf(`<span data-prop="%s"%s data-kind="number" data-source="%s"></span>`, name, lab, esc(val))
	case "bool":
		on, _ := v.(bool)
		return fmt.Sprintf(`<span data-prop="%s"%s data-kind="bool" data-source="%t"></span>`, name, lab, on)
	case "enum", "ref":
		// A choice carries the value it stores; the options name it.
		raw, _ := v.(string)
		options := s.choices(f, raw)
		val = raw
		if options == "" {
			// A ref past the most a list can hold is changed by asking.
			return ""
		}
		if !f.Required {
			options = strings.Replace(options, `data-options="[`, `data-options="[{&#34;value&#34;:&#34;&#34;,&#34;label&#34;:&#34;None&#34;},`, 1)
		}
		return fmt.Sprintf(`<span data-prop="%s"%s data-source="%s"%s>%s</span>`, name, lab, esc(val), options, esc(val))
	}
	// The editor reads what is there from data-source; the page already
	// says it where it shows, once.
	return fmt.Sprintf(`<span data-prop="%s"%s data-source="%s"></span>`, name, lab, esc(val))
}

// editControls is one of each control the design system has for a field,
// in a template the page does not show: the inline editor copies the one a
// field needs and fills in its name, value and choices, so every field it
// makes is the component itself, never markup of its own. Only on a page
// where something can be edited.
func (s *Server) editControls(parts ...template.HTML) template.HTML {
	editable := false
	for _, p := range parts {
		editable = editable || strings.Contains(string(p), "data-prop=") || strings.Contains(string(p), "data-edit-fields")
	}
	if !editable {
		return ""
	}
	control := func(kind, name string, props map[string]any) string {
		props["label"], props["name"], props["id"] = "Field", "field", "sw-control-"+kind
		return `<div data-control="` + kind + `">` + string(s.component(name, props)) + `</div>`
	}
	return template.HTML(`<template id="sw-controls">` +
		control("text", "text-field", map[string]any{"type": "text"}) +
		control("number", "text-field", map[string]any{"type": "number"}) +
		control("when", "text-field", map[string]any{"type": "text", "hint": "A day, like 19 Sep or next Friday, with a time if there is one, like 2pm."}) +
		control("textarea", "textarea", map[string]any{"rows": 3}) +
		control("select", "select", map[string]any{"options": []any{map[string]any{"value": "", "label": ""}}}) +
		control("checkbox", "checkbox", map[string]any{"value": "true"}) +
		`</template>`)
}
