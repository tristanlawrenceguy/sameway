package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// An arrangement is a page's worth of thought applied in one call: the
// blocks a job wants, where each sits and how wide, in the order they
// should arrive. Every block shows the person's own records, so the page
// holds what is theirs or says there is nothing yet; it is never a page of
// example words for the person to clear away (the evaluation's models
// left "Notes for the week go here." and "Your first book" on pages).

var arrangementOp = Op{Title: "Add a ready-made arrangement",
	Words: []string{"arrangement", "layout", "arrange"},
	Doing: saying("Arranging the page"),
	Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.addArrangement(a.Name, a.Fills) },
	Tool: llm.Tool{
		Name:        "add_arrangement",
		Description: "Add a whole arrangement of blocks for a job the person named, laid out as the catalogue says, in one call. Each block shows the person's own records as they are (their tasks, events, notes, habits), kept current, and says so on the page when there are none yet; nothing in it is example text. The result says what each block shows. To put things on it, create the records the person gave you with create_record; never invent any. An arrangement that needs a type the workspace lacks adds nothing and says how to make it.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{
			"name":  map[string]any{"type": "string", "description": "An arrangement from the catalogue."},
			"fills": map[string]any{"type": "object", "description": "Props to change on a block, by block key, such as its label or conditions: {\"todo\": {\"label\": \"Due soon\"}}. What a block lists comes from records and cannot be filled in."},
		}, "required": []string{"name"}, "additionalProperties": false},
	}, Offered: func(s *Service, t *llm.Tool) bool {
		// The arrangements as the catalogue has them now.
		var names []string
		for _, a := range s.Registry.Arrangements() {
			names = append(names, a.Name)
		}
		withProp(t, "name", map[string]any{"type": "string", "enum": names, "description": "An arrangement from the catalogue."})
		return true
	}}

// addArrangement adds each block in order, on the tab the person is
// looking at, and reports every one so each arrives and can be undone on
// its own.
func (s *Service) addArrangement(name string, fills map[string]any) toolResult {
	a, ok := s.Registry.Arrangement(name)
	if !ok {
		var names []string
		for _, x := range s.Registry.Arrangements() {
			names = append(names, x.Name)
		}
		return fail("no arrangement %q; the arrangements are %s", name, strings.Join(names, ", "))
	}
	if missing := s.missingNeeds(a); missing != "" {
		return fail("nothing added: %s", missing)
	}
	var out toolResult
	var lines []string
	// Every block is checked before any is added, so one that could not
	// be shown leaves no half of the page behind.
	all := make([]map[string]any, len(a.Blocks))
	for i, b := range a.Blocks {
		c, ok := s.Registry.Get(b.Component)
		if !ok {
			return fail("arrangement %s, block %s: unknown component %q", name, b.Key, b.Component)
		}
		props := map[string]any{}
		for k, v := range b.Props {
			props[k] = v
		}
		if fill, ok := fills[b.Key].(map[string]any); ok {
			for k, v := range fill {
				if serverFilled(c, k) {
					typeName, _ := props["type"].(string)
					return fail("arrangement %s, block %s: %s is filled in from the person's %s records, not given. Leave it out of fills; create_record makes each thing the person named, and it shows here", name, b.Key, k, orWord(typeName, "own"))
				}
				props[k] = v
			}
		}
		all[i] = props
		if _, bad := s.writable("add_arrangement", c, props); bad != nil {
			return fail("arrangement %s, block %s: %s", name, b.Key, bad.text)
		}
	}
	for i, b := range a.Blocks {
		l := look{Frame: b.Frame, Tone: b.Tone, Region: b.Region}
		if b.Span > 0 {
			span := b.Span
			l.Span = &span
		}
		r := s.addComponent(b.Component, all[i], l)
		if r.isErr {
			return fail("arrangement %s, block %s: %s", name, b.Key, r.text)
		}
		if r.change != nil {
			out.changes = append(out.changes, *r.change)
		}
		lines = append(lines, b.Key+": "+r.text)
	}
	out.text = fmt.Sprintf("added the %s arrangement, %d blocks, top to bottom:\n%s\nEach block shows the person's own records as they are and keeps current; one that shows nothing yet is right while they have none, and says so on the page. To fill it, create_record the things the person named; do not make any up.", name, len(a.Blocks), strings.Join(lines, "\n"))
	return out
}

// missingNeeds says which content type or field an arrangement shows that
// the workspace does not have, and the call that makes it; "" when all is
// there. Of a type that is there, only the fields its blocks pick records
// by are looked for: a book type with no author still shows its books.
func (s *Service) missingNeeds(a *render.Arrangement) string {
	types := s.Store.Types()
	for _, n := range a.Needs {
		t, ok := types.Get(n.Type)
		if !ok {
			call, _ := json.Marshal(map[string]any{"name": n.Type, "description": n.Description, "properties": n.Fields})
			return fmt.Sprintf("the %s arrangement shows %s records, and the workspace has no %s type. If the person wants it, make the type with add_type %s, then call add_arrangement again; add a record for each %s the person names, and none they did not", a.Name, n.Type, n.Type, call, n.Type)
		}
		used := pickedBy(a, n.Type)
		for _, f := range n.Fields {
			field, _ := f["name"].(string)
			if _, ok := t.Field(field); ok || !used[field] {
				continue
			}
			def := map[string]any{"type": n.Type}
			for k, v := range f {
				def[k] = v
			}
			call, _ := json.Marshal(def)
			return fmt.Sprintf("the %s arrangement shows %s records by their %s, and %s has no %s field. Add it with add_field %s, then call add_arrangement again", a.Name, n.Type, field, n.Type, field, call)
		}
	}
	return ""
}

// pickedBy is the fields an arrangement's blocks of a type pick and order
// their records by: status in status=reading, due in -due.
func pickedBy(a *render.Arrangement, typeName string) map[string]bool {
	out := map[string]bool{}
	for _, b := range a.Blocks {
		if b.Props["type"] != typeName {
			continue
		}
		where, _ := b.Props["where"].([]any)
		for _, w := range where {
			cond, _ := w.(string)
			if i := strings.IndexAny(cond, "=<>!~"); i > 0 {
				out[cond[:i]] = true
			}
		}
		if order, _ := b.Props["order"].(string); order != "" {
			out[strings.TrimPrefix(order, "-")] = true
		}
	}
	return out
}

// serverFilled is whether a prop is one the page fills in from records,
// such as a collection's items: given in fills, it would be thrown away
// when the block is drawn, and the person's words with it.
func serverFilled(c *render.Component, prop string) bool {
	var schema struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if json.Unmarshal(c.Manifest.Props, &schema) != nil {
		return false
	}
	return strings.Contains(schema.Properties[prop].Description, "Filled in by the server")
}

func orWord(s, instead string) string {
	if s == "" {
		return instead
	}
	return s
}

// arrangementCatalogue is the thought behind whole pages, for the prompt.
func (s *Service) arrangementCatalogue() string {
	arrangements := s.Registry.Arrangements()
	if len(arrangements) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nArrangements (name: description; when; then the blocks by key as component, region, span). When the person asks for a whole thing rather than one block, add_arrangement lays it out in one call. Its blocks show the person's own records and say so when there are none; fill them by creating the records the person gave you, never examples:\n")
	for _, a := range arrangements {
		fmt.Fprintf(&b, "\n%s: %s", a.Name, a.Description)
		if a.Use != nil {
			fmt.Fprintf(&b, " Use when: %s", a.Use.When)
			if a.Use.Not != "" {
				fmt.Fprintf(&b, " Not when: %s", a.Use.Not)
			}
		}
		for _, n := range a.Needs {
			fmt.Fprintf(&b, " Needs a %s type.", n.Type)
		}
		b.WriteString("\n")
		for _, blk := range a.Blocks {
			region := blk.Region
			if region == "" {
				region = "main"
			}
			span := blk.Span
			if span == 0 {
				span = 6
			}
			what := blk.Component
			if t, _ := blk.Props["type"].(string); t != "" {
				what += " of " + t
				if w, _ := blk.Props["where"].([]any); len(w) > 0 {
					what += fmt.Sprintf(" %v", w)
				}
			}
			fmt.Fprintf(&b, "  %s: %s, %s, span %d\n", blk.Key, what, region, span)
		}
	}
	return b.String()
}
