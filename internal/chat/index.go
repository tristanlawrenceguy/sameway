package chat

import (
	"fmt"
	"github.com/tristanlawrenceguy/sameway/internal/jsontext"
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
	if u := c.Manifest.Use; u != nil && u.When != "" {
		fmt.Fprintf(&b, "When it serves a person: %s\n", u.When)
	}
	fmt.Fprintf(&b, "Props schema: %s", ForModel(c.Manifest.Props))
	if len(c.Manifest.Examples) > 0 {
		ex, _ := jsontext.Marshal(c.Manifest.Examples[0].Props)
		fmt.Fprintf(&b, "\nExample props: %s", ex)
	}
	return b.String()
}

// typeHelp is the display names of a type's writable fields, for a refusal.
func typeHelp(t *schema.Type) string {
	return fmt.Sprintf("%s takes, inside fields: %s.", t.Name, strings.Join(typeFields(t), ", "))
}

// typeFields returns the display names of fields a model may write.
func typeFields(t *schema.Type) []string {
	var fields []string
	for _, f := range t.Shown() {
		if f.ReadOnly || schema.ServerFilled(f.Description) {
			continue
		}
		fields = append(fields, f.Display())
	}
	return fields
}

// componentIndex is each component in a line: what it is. When it serves
// a person, its props and an example come with a refusal (componentHelp),
// for the one it tries, not all fifty every turn.
func (s *Service) componentIndex() string {
	var b strings.Builder
	b.WriteString("\n\nComponents (name: what it is). A refusal gives when one serves a person and the props it takes:\n")
	for _, c := range s.Registry.Blocks() {
		fmt.Fprintf(&b, "%s: %s\n", c.Manifest.Name, firstSentence(c.Manifest.Description))
	}
	return b.String()
}

// typeIndex is each content type in a line: what it is, its fields by
// name and kind, and how much it holds (type_use.go).
func (s *Service) typeIndex() string {
	types := records.ContentTypes(s.Store)
	if len(types) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nContent types (name: what it is; its fields; how many records it holds). A record of one is what the person finds at /t/<type>; make it with create_record, never as a card on the canvas. A type with no records that an agent or the assistant made clutters the person's menus: you may ask them whether to hide it (change_field hide keeps everything), but never hide one without their yes, since it is their workspace:\n")
	makers := s.lazyMakers()
	for _, t := range types {
		fmt.Fprintf(&b, "%s: %s Fields: %s.%s\n", t.Name, firstSentence(t.Description), strings.Join(typeFields(t), ", "), s.typeUse(t, makers)+s.twinsSaid(t)) // type_use.go, type_twins.go
	}
	return b.String()
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
