package chat

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// humanizeValidationError replaces JSON Schema validation error syntax with
// plain-language text, for a record's fields; a component's props are said
// by PropsTrouble.
func humanizeValidationError(err string) string {
	err = regexp.MustCompile(`(?i)additional properties .* not allowed`).ReplaceAllString(err, "something I don't recognise")
	return err
}

// blockFields are what a block has beside its props: where it sits and how
// it looks. Models put them inside props more than any other mistake.
var blockFields = map[string]bool{"span": true, "frame": true, "tone": true, "region": true, "size": true, "canvas": true, "position": true}

// sameThing are names models use for one thing, the likeliest first: a
// button's words are its label, and a model sends text.
var sameThing = [][]string{
	{"label", "text", "title", "name", "caption", "heading", "content", "message", "body", "value"},
	{"href", "url", "link", "to", "path"},
	{"items", "rows", "entries", "list", "options", "choices", "values"},
	{"kind", "variant", "style", "look"},
	{"level", "depth"},
}

// PropsTrouble is props that do not fit a component, said for whoever has
// to fix the call, a model or a program: each prop that is wrong and what
// to do about it, where a block field belongs, the prop likely meant, what
// is allowed, and every prop the component takes. A person reads the
// error's own words (render.PropsError), which name no schema; this names
// everything, since it is read to be acted on. An error that is not about
// props is its own words.
func PropsTrouble(err error) string {
	var pe *render.PropsError
	if !errors.As(err, &pe) {
		return err.Error()
	}
	c := pe.Component
	var lines []string
	for _, p := range pe.Problems {
		name := p.Prop
		if p.At != "" {
			name = p.At + "/" + p.Prop
		}
		switch p.Kind {
		case "unknown":
			switch {
			case p.At == "" && blockFields[p.Prop]:
				lines = append(lines, fmt.Sprintf("%s is a block field, not a %s prop: pass it beside props, not inside them", p.Prop, c))
			case p.At == "":
				line := fmt.Sprintf("%s is not a %s prop", p.Prop, c)
				if near := nearestProp(p.Prop, pe.Props); near != "" {
					line += fmt.Sprintf("; %s takes %s, not %s", c, near, p.Prop)
				}
				lines = append(lines, line)
			default:
				lines = append(lines, fmt.Sprintf("%s is not something %s takes", p.Prop, p.At))
			}
		case "missing":
			lines = append(lines, fmt.Sprintf("%s is required and missing", name))
		case "enum":
			lines = append(lines, fmt.Sprintf("%s must be one of %s, not %s", name, strings.Join(p.Allowed, ", "), p.Got))
		case "type":
			lines = append(lines, fmt.Sprintf("%s must be %s, not %s", name, strings.Join(p.Allowed, " or "), withA(p.Got)))
		default:
			if name == "" {
				name = "props"
			}
			lines = append(lines, name+": "+p.Words)
		}
	}
	var props []string
	required := map[string]bool{}
	for _, r := range pe.Required {
		required[r] = true
	}
	for _, p := range pe.Props {
		if required[p] {
			p += " (required)"
		}
		props = append(props, p)
	}
	return fmt.Sprintf("%s props do not fit, so nothing was saved:\n- %s\n%s takes: %s.", c, strings.Join(lines, "\n- "), c, strings.Join(props, ", "))
}

// nearestProp is the prop a wrong name was likely meant to be: one of the
// same thing by another name, else a slip of a letter or two.
func nearestProp(name string, props []string) string {
	has := map[string]bool{}
	for _, p := range props {
		has[p] = true
	}
	for _, group := range sameThing {
		in := false
		for _, n := range group {
			in = in || n == strings.ToLower(name)
		}
		if !in {
			continue
		}
		for _, n := range group {
			if has[n] && n != name {
				return n
			}
		}
	}
	return render.Nearest(name, props)
}

// withA is a JSON type with its article: a string, an object, an array.
func withA(kind string) string {
	if kind == "" {
		return "that"
	}
	if strings.ContainsRune("aeiou", rune(kind[0])) {
		return "an " + kind
	}
	return "a " + kind
}
