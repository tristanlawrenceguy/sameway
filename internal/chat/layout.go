package chat

import (
	"fmt"
	"slices"
	"strings"
)

// look is how a block sits on the canvas: its width, its order, and how
// much surface it brings. All optional, all independent of its props.
type look struct {
	Span, Position            *int
	Frame, Tone, Region, Size string
	// Canvas is the tab the block sits on, as a canvas id; "" is Home. Only
	// applied when the caller says so, since "" also means "not given".
	Canvas    string
	SetCanvas bool
}

// noCanvas is the refusal of a tab that is not there.
func noCanvas(id string) error {
	return fmt.Errorf("no canvas with id %q; the tabs and their ids are listed in the prompt, and \"\" is Home", id)
}

// place is apply for a call that may name a tab, which must be there.
func (s *Service) place(l look, fields map[string]any) ([]string, error) {
	if l.SetCanvas && !s.HasCanvas(l.Canvas) {
		return nil, noCanvas(l.Canvas)
	}
	return l.apply(fields)
}

// What a block's frame, tone, region and size may be, for the tools'
// schemas and for apply.
var (
	frames  = []string{"card", "bare"}
	tones   = []string{"none", "accent", "success", "warning", "danger", "info"}
	regions = []string{"main", "left", "right", "header", "footer"}
	sizes   = []string{"full", "compact", "icon"}
)

// apply writes the layout fields that were given, and describes them.
func (l look) apply(fields map[string]any) ([]string, error) {
	var what []string
	set := func(field string, v any) {
		fields[field] = v
		what = append(what, fmt.Sprintf("%s %v", field, v))
	}
	if l.Span != nil {
		if *l.Span < 1 || *l.Span > 12 {
			return nil, fmt.Errorf("span must be between 1 and 12, got %d", *l.Span)
		}
		set("span", *l.Span)
	}
	if l.Position != nil {
		set("position", *l.Position)
	}
	for _, c := range []struct {
		field, got string
		allowed    []string // nil for tone, which the block's own schema checks
	}{{"frame", l.Frame, frames}, {"tone", l.Tone, nil}, {"region", l.Region, regions}, {"size", l.Size, sizes}} {
		if c.got == "" {
			continue
		}
		if c.allowed != nil && !slices.Contains(c.allowed, c.got) {
			return nil, fmt.Errorf("%s must be one of %s, got %q", c.field, strings.Join(c.allowed, ", "), c.got)
		}
		set(c.field, c.got)
	}
	if l.SetCanvas {
		fields["canvas"] = l.Canvas
		what = append(what, "canvas "+orWord(l.Canvas, "Home"))
	}
	return what, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
