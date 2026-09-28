package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// PropsError is props that do not fit a component's schema, kept whole so
// each reader gets it in their own words. Error is for a person: a line
// per problem, by the prop's readable name, with no schema words. Problems
// and Props are for whoever has to fix the call, a model or a program,
// which needs the exact prop, what is allowed and what there is instead;
// see chat.PropsTrouble for those words.
type PropsError struct {
	Component string
	Problems  []PropProblem
	// Props is every prop the component takes, by name, and Required
	// those it cannot do without.
	Props, Required []string
	person          []string
}

// PropProblem is one thing wrong with the props.
type PropProblem struct {
	// At is where, for a prop inside another ("items/0"); empty at the
	// top, which is where most are.
	At string
	// Prop is the prop the problem is about.
	Prop string
	// Kind is unknown (no such prop), missing (required and not given),
	// enum (not one of Allowed), type (not the kind of value Allowed
	// names) or other (Words says what).
	Kind    string
	Allowed []string
	Got     string
	Words   string
}

func (e *PropsError) Error() string { return strings.Join(e.person, "; ") }

// propsError turns a validator error into a PropsError: one person's line
// per problem, as the canvas has always said them, and a problem per prop
// for the one fixing the call.
func propsError(ps *propSchema, ve *jsonschema.ValidationError) *PropsError {
	printer := message.NewPrinter(language.English)
	out := &PropsError{Component: ps.name, Required: ps.required}
	for name := range ps.properties {
		out.Props = append(out.Props, name)
	}
	sort.Strings(out.Props)
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) > 0 {
			for _, c := range e.Causes {
				walk(c)
			}
			return
		}
		words := e.ErrorKind.LocalizedString(printer)
		var at []string
		for _, seg := range e.InstanceLocation {
			if seg != "" {
				at = append(at, seg)
			}
		}
		// A problem with a value is about the prop it is in; one with an
		// object (unknown, missing) is about props inside it.
		prop, where := "", strings.Join(at, "/")
		if len(at) > 0 {
			prop, where = at[len(at)-1], strings.Join(at[:len(at)-1], "/")
		}
		switch k := e.ErrorKind.(type) {
		case *kind.AdditionalProperties:
			words = "something I don't recognise"
			for _, p := range k.Properties {
				out.Problems = append(out.Problems, PropProblem{At: strings.Join(at, "/"), Prop: p, Kind: "unknown"})
			}
		case *kind.Required:
			for _, p := range k.Missing {
				out.Problems = append(out.Problems, PropProblem{At: strings.Join(at, "/"), Prop: p, Kind: "missing"})
			}
		case *kind.Enum:
			allowed := make([]string, 0, len(k.Want))
			for _, w := range k.Want {
				allowed = append(allowed, jsonWords(w))
			}
			out.Problems = append(out.Problems, PropProblem{At: where, Prop: prop, Kind: "enum", Allowed: allowed, Got: jsonWords(k.Got)})
		case *kind.Type:
			out.Problems = append(out.Problems, PropProblem{At: where, Prop: prop, Kind: "type", Allowed: k.Want, Got: k.Got})
		default:
			out.Problems = append(out.Problems, PropProblem{At: where, Prop: prop, Kind: "other", Words: words})
		}
		out.person = append(out.person, fmt.Sprintf("%s: %s", locationLabel(e.InstanceLocation, ps, e), words))
	}
	walk(ve)
	return out
}

// jsonWords is a value as it is written in JSON: "info", 3, true.
func jsonWords(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(raw)
}
