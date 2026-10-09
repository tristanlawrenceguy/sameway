// Package ui is the components the pages use most, as Go types: a button,
// a link, an alert, a text field, an empty state, a status and a mark, and
// a form that posts an action. Each is written here by hand, not generated:
// its fields carry the prop names of the component's manifest, and a test
// holds every field to that manifest (names, required props, enums), so
// the two cannot drift apart. The manifest stays the one definition of the
// props: what a builder gives is still a map, validated and given its
// defaults when it is rendered (internal/render).
//
// A builder is for what a page says often. A component a page shows once,
// with props no other page shares, stays a map where it is used.
package ui

import (
	"html/template"
	"reflect"
	"strings"
)

// Part is a component with its props, ready to render.
type Part interface {
	// Component is the name of the component's folder.
	Component() string
	// Props are the props to render it with: every field that is set,
	// under its prop name. A field left at its zero value is left out, so
	// the manifest's default applies.
	Props() map[string]any
}

// Renderer renders a part, as the server does with its registry.
type Renderer func(Part) template.HTML

// Bool is a true or false for a prop whose default is not false, such as
// a text field's spellcheck or a button's pressed, where leaving it out
// and saying false are different things.
func Bool(b bool) *bool { return &b }

// props reads a builder's fields into a map by their prop tags, leaving
// out the ones at their zero value. A nested struct becomes a map of its
// own, as the empty state's action does.
func props(v any) map[string]any {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	out := map[string]any{}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("prop"), ",")
		f := rv.Field(i)
		if name == "" || f.IsZero() {
			continue
		}
		if f.Kind() == reflect.Pointer {
			f = f.Elem()
		}
		if f.Kind() == reflect.Struct {
			out[name] = props(f.Interface())
			continue
		}
		out[name] = f.Interface()
	}
	return out
}
