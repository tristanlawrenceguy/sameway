package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
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
	// lines splits text on single newlines.
	"lines": func(s string) []string {
		return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	},
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
	if props == nil {
		props = map[string]any{}
	}
	inst := roundTrip(props)
	if err := p.schema.Validate(inst); err != nil {
		return nil, formatValidation(err)
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
func formatValidation(err error) error {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	var lines []string
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			loc := "/" + strings.Join(e.InstanceLocation, "/")
			lines = append(lines, fmt.Sprintf("%s: %v", loc, e.ErrorKind))
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	return fmt.Errorf("invalid props: %s", strings.Join(lines, "; "))
}
