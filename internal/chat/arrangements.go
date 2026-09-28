package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// An arrangement is a page's worth of thought applied in one call: the
// blocks a job wants, where each sits and how wide, in the order they
// should arrive. The assistant fills in the person's words and adjusts
// afterwards with the ordinary tools; it does not compose a layout from
// nothing when one has already been thought through.

func (s *Service) arrangementTool() llm.Tool {
	var names []string
	for _, a := range s.Registry.Arrangements() {
		names = append(names, a.Name)
	}
	return llm.Tool{
		Name:        "add_arrangement",
		Description: "Add a whole arrangement of blocks for a job the person named, laid out as the catalogue says: the right components, regions and widths, in one call. Fill in their words with fills, keyed by block; adjust afterwards with update_component if needed.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{
			"name":  map[string]any{"type": "string", "enum": names, "description": "An arrangement from the catalogue."},
			"fills": map[string]any{"type": "object", "description": "Props to put in place of the starting ones, by block key: {\"todo\": {\"items\": [\"...\"]}}."},
		}, "required": []string{"name"}, "additionalProperties": false},
	}
}

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
	var out toolResult
	var lines []string
	for _, b := range a.Blocks {
		props := map[string]any{}
		for k, v := range b.Props {
			props[k] = v
		}
		if fill, ok := fills[b.Key].(map[string]any); ok {
			for k, v := range fill {
				props[k] = v
			}
		}
		l := look{Frame: b.Frame, Tone: b.Tone, Region: b.Region}
		if b.Span > 0 {
			span := b.Span
			l.Span = &span
		}
		r := s.addComponent(b.Component, props, l)
		if r.isErr {
			return fail("arrangement %s, block %s: %s", name, b.Key, r.text)
		}
		if r.change != nil {
			out.changes = append(out.changes, *r.change)
		}
		lines = append(lines, b.Key+": "+r.text)
	}
	out.text = fmt.Sprintf("added the %s arrangement, %d blocks, top to bottom:\n%s", name, len(a.Blocks), strings.Join(lines, "\n"))
	return out
}

// arrangementCatalogue is the thought behind whole pages, for the prompt.
func (s *Service) arrangementCatalogue() string {
	arrangements := s.Registry.Arrangements()
	if len(arrangements) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nArrangements (name: description; when; then the blocks by key as component, region, span). When the person asks for a whole thing rather than one block, add_arrangement gives a considered layout in one call, with fills for their words:\n")
	for _, a := range arrangements {
		fmt.Fprintf(&b, "\n%s: %s", a.Name, a.Description)
		if a.Use != nil {
			fmt.Fprintf(&b, " Use when: %s", a.Use.When)
			if a.Use.Not != "" {
				fmt.Fprintf(&b, " Not when: %s", a.Use.Not)
			}
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
			fmt.Fprintf(&b, "  %s: %s, %s, span %d\n", blk.Key, blk.Component, region, span)
		}
	}
	return b.String()
}
