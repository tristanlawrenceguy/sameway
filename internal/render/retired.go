package render

// retired names components no longer offered, each with the one that now
// does its job and how a saved block's props carry across. A canvas saved
// before a component was retired keeps showing what it held; the catalogue,
// the describe output and the tool schemas list only r.byName, so the
// assistant cannot choose a retired one again.
var retired = map[string]struct {
	to    string
	props func(map[string]any) map[string]any
}{
	// datepicker was a date input with no words: the when-field asks in
	// words and says back how it read them, with the same picker beside.
	// Its ISO value is both the words (the server reads YYYY-MM-DD) and the
	// day the picker opens on; min and max have no place in a when-field.
	"datepicker": {to: "when-field", props: func(p map[string]any) map[string]any {
		out := map[string]any{}
		for _, k := range []string{"label", "name", "id", "hint", "error", "required"} {
			if v, ok := p[k]; ok {
				out[k] = v
			}
		}
		if v, ok := p["value"].(string); ok && v != "" {
			out["value"], out["day"] = v, v
		}
		return out
	}},
}

// Resolve returns the component a block names, with its props. A name the
// registry has always wins, so a workspace may still define its own
// component by a retired name; a retired built-in resolves to the one that
// replaced it, with its props carried across.
func (r *Registry) Resolve(name string, props map[string]any) (*Component, map[string]any, bool) {
	if c, ok := r.byName[name]; ok {
		return c, props, true
	}
	if old, ok := retired[name]; ok {
		if c, ok := r.byName[old.to]; ok {
			return c, old.props(props), true
		}
	}
	return nil, props, false
}
