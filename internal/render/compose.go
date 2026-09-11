package render

import (
	"fmt"
	"html/template"
)

// Composition: a component can place other components inside itself.
//
// A prop may carry a spec, {"component": "button", "props": {...}}, and the
// template renders it with {{child .action}}. The child is validated against
// its own manifest exactly like any other render, so a calendar event can
// carry real buttons rather than markup copied out of the button component.
//
// Nesting is bounded. Each component's template is parsed once per depth,
// and the deepest copy has no child function left to call, so a component
// that contained itself would stop rather than recurse forever. Three is
// enough for the arrangements a page actually needs, and a hard stop beats
// a stack overflow served to a person.
const maxNesting = 3

// funcsAt returns the template functions for a component parsed to sit at
// the given nesting depth.
func (r *Registry) funcsAt(depth int) template.FuncMap {
	funcs := template.FuncMap{}
	for name, fn := range Funcs {
		funcs[name] = fn
	}
	funcs["child"] = func(spec any) template.HTML { return r.child(spec, depth) }
	funcs["children"] = func(specs any) template.HTML {
		list, ok := specs.([]any)
		if !ok {
			return ""
		}
		var out template.HTML
		for _, s := range list {
			out += r.child(s, depth)
		}
		return out
	}
	return funcs
}

// child renders one nested component spec.
func (r *Registry) child(spec any, depth int) template.HTML {
	if spec == nil {
		return ""
	}
	m, ok := spec.(map[string]any)
	if !ok {
		return problem("a nested component must be an object with component and props")
	}
	name, _ := m["component"].(string)
	if name == "" {
		return problem("a nested component needs a component name")
	}
	if depth+1 >= maxNesting {
		return problem(fmt.Sprintf("%s is nested too deeply; components may sit %d levels inside each other", name, maxNesting))
	}
	c, ok := r.byName[name]
	if !ok {
		return problem(fmt.Sprintf("unknown component %q", name))
	}
	props, _ := m["props"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	out, err := c.renderAt(props, nil, depth+1)
	if err != nil {
		return problem(err.Error())
	}
	return out
}

// problem renders a fault where the component would have been, so a bad
// spec is visible on the page rather than silently missing.
func problem(msg string) template.HTML {
	return template.HTML(`<span class="sw-problem" role="status">` + template.HTMLEscapeString(msg) + `</span>`)
}
