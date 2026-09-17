package chat

import "fmt"

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

// apply writes the layout fields that were given, and describes them.
func (l look) apply(fields map[string]any) ([]string, error) {
	var what []string
	if l.Span != nil {
		if *l.Span < 1 || *l.Span > 12 {
			return nil, fmt.Errorf("span must be between 1 and 12, got %d", *l.Span)
		}
		fields["span"] = *l.Span
		what = append(what, fmt.Sprintf("span %d", *l.Span))
	}
	if l.Position != nil {
		fields["position"] = *l.Position
		what = append(what, fmt.Sprintf("position %d", *l.Position))
	}
	if l.Frame != "" {
		if l.Frame != "card" && l.Frame != "bare" {
			return nil, fmt.Errorf("frame must be card or bare, got %q", l.Frame)
		}
		fields["frame"] = l.Frame
		what = append(what, "frame "+l.Frame)
	}
	if l.Tone != "" {
		fields["tone"] = l.Tone
		what = append(what, "tone "+l.Tone)
	}
	if l.Region != "" {
		switch l.Region {
		case "main", "left", "right", "header", "footer":
		default:
			return nil, fmt.Errorf("region must be main, left, right, header, or footer, got %q", l.Region)
		}
		fields["region"] = l.Region
		what = append(what, "region "+l.Region)
	}
	if l.Size != "" {
		switch l.Size {
		case "full", "compact", "icon":
		default:
			return nil, fmt.Errorf("size must be full, compact, or icon, got %q", l.Size)
		}
		fields["size"] = l.Size
		what = append(what, "size "+l.Size)
	}
	if l.SetCanvas {
		fields["canvas"] = l.Canvas
		if l.Canvas == "" {
			what = append(what, "canvas Home")
		} else {
			what = append(what, "canvas "+l.Canvas)
		}
	}
	return what, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
