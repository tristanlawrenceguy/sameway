package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The assistant is told what there is in a line each, and reads the whole
// of one thing when it needs it: a component's props and an example, a
// type's fields as a schema, an arrangement's blocks. A few steps that
// fetch the right thing beat sending everything every turn, which grew
// the prompt past 115 KB and left no room for anything new.

var detailsTool = llm.Tool{
	Name:        "details",
	Description: "Read one thing in full before you use it: a component's props with an example (before add_component or update_component), a content type's fields as a schema (before create_record or update_record on it), or an arrangement's blocks. The prompt lists each in a line; this gives the rest. Read what you have not read this conversation; a refusal also says what it takes.",
	Schema: obj(map[string]any{
		"name": map[string]any{"type": "string", "description": "The component, content type or arrangement, as the prompt lists it."},
	}, "name"),
}

func (s *Service) details(name string) toolResult {
	name = strings.ToLower(strings.TrimSpace(name))
	if c, ok := s.Registry.Get(name); ok && !c.Manifest.PageOnly {
		var b strings.Builder
		fmt.Fprintf(&b, "component %s: %s\n", c.Manifest.Name, c.Manifest.Description)
		if u := c.Manifest.Use; u != nil {
			fmt.Fprintf(&b, "Use when: %s", u.When)
			if u.Not != "" {
				fmt.Fprintf(&b, " Not when: %s", u.Not)
			}
			if u.With != "" {
				fmt.Fprintf(&b, " With: %s", u.With)
			}
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Props schema: %s\n", ForModel(c.Manifest.Props))
		if len(c.Manifest.Examples) > 0 {
			ex, _ := json.Marshal(c.Manifest.Examples[0].Props)
			fmt.Fprintf(&b, "Example props: %s\n", ex)
		}
		return toolResult{text: b.String()}
	}
	for _, t := range s.contentTypes() {
		if t.Name == name {
			sch := t.JSONSchema()
			delete(sch, "description")
			raw, _ := json.Marshal(sch)
			return toolResult{text: fmt.Sprintf("content type %s: %s\nFields schema: %s\n", t.Name, t.Description, ForModel(raw))}
		}
	}
	if text := s.arrangementDetails(name); text != "" {
		return toolResult{text: text}
	}
	return fail("nothing is called %q; the prompt lists the components, content types and arrangements by name", name)
}

// componentIndex is each component in a line: what it is, and when it
// serves a person.
func (s *Service) componentIndex() string {
	var b strings.Builder
	b.WriteString("\n\nComponents (name: what it is; when it serves a person). Read one with details before you add it:\n")
	for _, c := range s.Registry.Blocks() {
		fmt.Fprintf(&b, "%s: %s", c.Manifest.Name, firstSentence(c.Manifest.Description))
		if u := c.Manifest.Use; u != nil && u.When != "" {
			fmt.Fprintf(&b, " When: %s", firstSentence(u.When))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// typeIndex is each content type in a line: what it is, and its fields
// by name and kind; details gives the schema.
func (s *Service) typeIndex() string {
	types := s.contentTypes()
	if len(types) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nContent types (name: what it is; its fields). A record of one is what the person finds at /t/<type>; make it with create_record, never as a card on the canvas, and read its schema with details first:\n")
	for _, t := range types {
		var fields []string
		for _, f := range t.Shown() {
			if f.ReadOnly || ServerFilled(f.Description) {
				continue
			}
			fields = append(fields, f.Name+" ("+fieldKind(f)+")")
		}
		fmt.Fprintf(&b, "%s: %s Fields: %s\n", t.Name, firstSentence(t.Description), strings.Join(fields, ", "))
	}
	return b.String()
}

// fieldKind is a field's kind in a word or two.
func fieldKind(f schema.Field) string {
	req := ""
	if f.Required {
		req = ", required"
	}
	switch {
	case f.RefList():
		return "ids of " + f.To + req
	case f.Type == "ref":
		return "id of " + f.To + req
	case f.Type == "enum":
		vals := append([]string{}, f.Values...)
		sort.Strings(vals)
		return strings.Join(f.Values, "|") + req
	case f.Type == "datetime":
		return "date" + req
	case f.Type == "markdown", f.Type == "text", f.Type == "string":
		return "text" + req
	case f.Type == "bool":
		return "yes/no" + req
	case f.Type == "int", f.Type == "float":
		return "number" + req
	}
	return f.Type + req
}

// firstSentence is a description's first sentence.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '.' && (s[i+1] == ' ' || s[i+1] == '\n') {
			return s[:i+1]
		}
	}
	return s
}

// arrangementDetails is one arrangement as the prompt says it, or "".
func (s *Service) arrangementDetails(name string) string {
	all := s.arrangementCatalogue()
	at := strings.Index(all, "\n"+name+": ")
	if at < 0 {
		return ""
	}
	rest := all[at+1:]
	if end := strings.Index(rest, "\n\n"); end >= 0 {
		rest = rest[:end]
	}
	return "arrangement " + rest + "\n"
}
