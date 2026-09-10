package schema

// JSON Schema is how a content type describes itself to anything outside
// this package: the describe endpoint, the API docs, and the tools a model
// calls. It is generated from the same field list the store and the forms
// use, so the three can never disagree.

// JSONSchema renders the type as a JSON Schema object, used by describe,
// the API docs, and later the MCP tools.
func (t *Type) JSONSchema() map[string]any {
	props := map[string]any{}
	var required []string
	for _, f := range t.Fields {
		p := map[string]any{}
		switch f.Type {
		case "string", "text", "markdown", "datetime":
			p["type"] = "string"
		case "enum":
			p["type"] = "string"
			p["enum"] = f.Values
		case "int":
			p["type"] = "integer"
		case "float":
			p["type"] = "number"
		case "bool":
			p["type"] = "boolean"
		case "list":
			p["type"] = "array"
			p["items"] = map[string]any{"type": "string"}
		case "json":
			p["type"] = []string{"object", "array", "string", "number", "boolean", "null"}
		}
		if f.Description != "" {
			p["description"] = f.Description
		}
		if f.Default != nil {
			p["default"] = f.Default
		}
		if f.MaxLength > 0 {
			p["maxLength"] = f.MaxLength
		}
		if f.Type == "datetime" {
			p["format"] = "date-time"
		}
		props[f.Name] = p
		if f.Required {
			required = append(required, f.Name)
		}
	}
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	if t.Description != "" {
		s["description"] = t.Description
	}
	return s
}
