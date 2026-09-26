package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Funcs are the template functions available to every component template.
var Funcs = template.FuncMap{
	// paragraphs splits text on blank lines so templates can emit one p per paragraph.
	"paragraphs": func(s string) []string {
		var out []string
		for _, part := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n\n") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
		return out
	},
	// optValue and optLabel let a select take either plain strings or
	// {value, label} objects, so what is stored can differ from what a
	// person reads without every caller needing two shapes.
	// markdown renders structured text: headings from the given level, lists,
	// emphasis, links, code, captioned tables. See internal/prose.
	"markdown": func(s string, base any) template.HTML { return prose.Render(s, num(base)) },
	// add is for a heading level one below another: the prose inside a
	// card starts a level under the card's own title.
	"add":      func(a, b any) int { return num(a) + num(b) },
	"optValue": func(v any) string { return optionPart(v, "value") },
	// json writes a value for a script to read from an attribute, such as
	// the choices the inline editor offers.
	"json":     func(v any) string { b, _ := json.Marshal(v); return string(b) },
	"optLabel": func(v any) string { return optionPart(v, "label") },
	// Calendar shape, computed here because a template cannot do date maths
	// and a month view must not need JavaScript. See calendar.go.
	"monthWeeks":   monthWeeks,
	"monthName":    monthName,
	"weekdayNames": weekdayNames,
	"eventsOn":     eventsOn,
	"dayHours":     dayHours,
	"chartLabel":   chartLabel,
	"numberText":   numberText,
	"allDay":       allDay,
	"upcoming":     upcoming,
	"shortDate":    shortDate,
	"longDate":     longDate,
	// Chart shape, for the same reason: arithmetic a template cannot do,
	// for a picture that must not need JavaScript. See chart.go.
	"chartShape":   chartShape,
	"chartNarrow":  chartNarrow,
	"chartSummary": chartSummary,
	"sparkline":    sparkline,
	"chartLast":    chartLast,
	"chartTrend":   chartTrend,
	// lines splits text on single newlines.
	"lines": func(s string) []string {
		return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	},
	// linkify wraps internal paths like /t/note/abc and external URLs
	// (http:// or https://) in <a class="sw-link" href="…">…</a> tags so that
	// assistant chat replies can be navigated with one click.  javascript:
	// payloads are rejected; everything else passes through as escaped text.
	"linkify": linkify,
	// For the meter and pagination components: see helpers.go.
	"dict":    dict,
	"percent": percent,
	"atMost":  atMost,
	"pages":   pagesOf,
}

// optionPart reads one half of a select option, whichever shape it came in.
func optionPart(v any, part string) string {
	if m, ok := v.(map[string]any); ok {
		if s, ok := m[part].(string); ok && s != "" {
			return s
		}
		if s, ok := m["value"].(string); ok {
			return s
		}
		return ""
	}
	return fmt.Sprint(v)
}

// propSchema is a compiled JSON Schema plus the parsed property list used to
// fill defaults and zero values before rendering.
type propSchema struct {
	schema     *jsonschema.Schema
	properties map[string]map[string]any
	order      []string
}

func compileProps(name string, raw json.RawMessage) (*propSchema, error) {
	if len(raw) == 0 {
		return nil, errors.New("manifest has no props schema")
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	url := "sameway://components/" + name + "/props.json"
	if err := compiler.AddResource(url, doc); err != nil {
		return nil, err
	}
	sch, err := compiler.Compile(url)
	if err != nil {
		return nil, err
	}
	ps := &propSchema{schema: sch, properties: map[string]map[string]any{}}
	if m, ok := doc.(map[string]any); ok {
		if props, ok := m["properties"].(map[string]any); ok {
			for k, v := range props {
				if pm, ok := v.(map[string]any); ok {
					ps.properties[k] = pm
					ps.order = append(ps.order, k)
				}
			}
		}
	}
	return ps, nil
}

// normalize validates props and returns a copy with defaults and zero values
// filled in for every declared property, integers as int64, so templates can
// use every prop without nil checks.
func (p *propSchema) normalize(props map[string]any) (map[string]any, error) {
	// A nil prop means "not given": models send null, and Go code passes
	// values it may not have. Drop them before validation.
	given := map[string]any{}
	for k, v := range props {
		if v != nil {
			given[k] = v
		}
	}
	inst := roundTrip(given)
	if err := p.schema.Validate(inst); err != nil {
		return nil, formatValidation(p, err)
	}
	out := map[string]any{}
	for k, v := range inst.(map[string]any) {
		out[k] = v
	}
	for name, def := range p.properties {
		typ, _ := def["type"].(string)
		if _, present := out[name]; !present {
			if d, ok := def["default"]; ok {
				out[name] = plainNumber(d)
			} else {
				out[name] = zeroFor(typ)
			}
		}
		if typ == "integer" {
			if f, ok := out[name].(float64); ok {
				out[name] = int64(f)
			}
		}
	}
	return out, nil
}

// roundTrip converts arbitrary Go values (int, []string, structs) into the
// plain JSON shapes the validator and templates expect.
func roundTrip(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

// plainNumber converts json.Number values from the schema document into the
// int64 or float64 the templates compare against.
func plainNumber(v any) any {
	n, ok := v.(json.Number)
	if !ok {
		return v
	}
	if i, err := n.Int64(); err == nil {
		return i
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	return v
}

func zeroFor(typ string) any {
	switch typ {
	case "boolean":
		return false
	case "integer":
		return int64(0)
	case "number":
		return float64(0)
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return ""
	}
}

// formatValidation turns a validator error into one readable line per problem.
func formatValidation(ps *propSchema, err error) error {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	printer := message.NewPrinter(language.English)
	var lines []string
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			lines = append(lines, fmt.Sprintf("%s: %s", locationLabel(e.InstanceLocation, ps, e), e.ErrorKind.LocalizedString(printer)))
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	return errors.New(strings.Join(lines, "; "))
}

// capitalize returns the string with its first letter uppercased.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// num reads a number however JSON or Go handed it over.
func num(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}
