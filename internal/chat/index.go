package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The assistant is told what there is in a line each, and tries. A call
// that does not fit is refused with the whole of what it takes: a
// component's props schema and an example, a type's fields. What it needs
// arrives when it needs it, where sending everything every turn grew the
// prompt past 115 KB, and a step to read first was one more thing a
// model could skip.

// componentHelp is a component's props schema and an example, for a
// refusal.
func componentHelp(c *render.Component) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Props schema: %s", ForModel(c.Manifest.Props))
	if len(c.Manifest.Examples) > 0 {
		ex, _ := json.Marshal(c.Manifest.Examples[0].Props)
		fmt.Fprintf(&b, "\nExample props: %s", ex)
	}
	return b.String()
}

// typeHelp is a type's fields by name and kind, for a refusal.
func typeHelp(t *schema.Type) string {
	return fmt.Sprintf("%s takes, inside fields: %s.", t.Name, strings.Join(typeFields(t), ", "))
}

// typeFields is each field a model may write, by name and kind.
func typeFields(t *schema.Type) []string {
	var fields []string
	for _, f := range t.Shown() {
		if f.ReadOnly || ServerFilled(f.Description) {
			continue
		}
		fields = append(fields, f.Name+" ("+fieldKind(f)+")")
	}
	return fields
}

// componentIndex is each component in a line: what it is, and when it
// serves a person.
func (s *Service) componentIndex() string {
	var b strings.Builder
	b.WriteString("\n\nComponents (name: what it is; when it serves a person). A refusal gives the props one takes:\n")
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
// by name and kind.
func (s *Service) typeIndex() string {
	types := records.ContentTypes(s.Store)
	if len(types) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nContent types (name: what it is; its fields). A record of one is what the person finds at /t/<type>; make it with create_record, never as a card on the canvas:\n")
	for _, t := range types {
		fmt.Fprintf(&b, "%s: %s Fields: %s\n", t.Name, firstSentence(t.Description), strings.Join(typeFields(t), ", "))
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
