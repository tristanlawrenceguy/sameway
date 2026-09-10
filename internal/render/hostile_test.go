package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// payloads are the strings an untrusted model or person might send as props.
var payloads = []string{
	`<script>alert(1)</script>`,
	`" onmouseover="alert(1)`,
	`javascript:alert(1)`,
	`</p><img src=x onerror=alert(1)>`,
	`{{.secret}}`,
}

// TestHostilePropsNeverBecomeMarkup swaps every string in every example for
// each payload and asserts the output contains no script, no event handler
// attribute, and no javascript: URL. This is the guarantee that lets a model
// write props straight to the page.
func TestHostilePropsNeverBecomeMarkup(t *testing.T) {
	for _, c := range builtins(t).Components() {
		var schema struct {
			Properties map[string]struct {
				Enum []any `json:"enum"`
			} `json:"properties"`
		}
		json.Unmarshal(c.Manifest.Props, &schema)
		for _, ex := range c.Manifest.Examples {
			for _, payload := range payloads {
				props := poison(ex.Props, payload, func(key string) bool {
					return len(schema.Properties[key].Enum) > 0
				})
				out, err := c.Render(props)
				if err != nil {
					// Schema constraints such as minLength may reject a payload; that is fine.
					continue
				}
				assertInert(t, c.Manifest.Name+"/"+ex.Name, payload, string(out))
			}
		}
	}
}

// poison replaces every string leaf with payload, except enum-typed props.
func poison(v map[string]any, payload string, isEnum func(string) bool) map[string]any {
	out := map[string]any{}
	for k, val := range v {
		if isEnum(k) {
			out[k] = val
			continue
		}
		out[k] = poisonValue(val, payload)
	}
	return out
}

func poisonValue(v any, payload string) any {
	switch x := v.(type) {
	case string:
		return payload
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = poisonValue(item, payload)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, item := range x {
			out[k] = poisonValue(item, payload)
		}
		return out
	}
	return v
}

func assertInert(t *testing.T, where, payload, out string) {
	t.Helper()
	// Template syntax in data must survive as literal text, not execute.
	if strings.Contains(payload, "{{") && !strings.Contains(out, "{{.secret}}") {
		t.Errorf("%s: template syntax in a prop was interpreted: %s", where, out)
	}
	doc, err := htmltest.Parse(out)
	if err != nil {
		t.Errorf("%s: unparsable output: %v", where, err)
		return
	}
	doc.Walk(func(n *html.Node) {
		if n.Data == "script" || n.Data == "img" {
			t.Errorf("%s: payload %q produced a <%s> element:\n%s", where, payload, n.Data, out)
		}
		for _, a := range n.Attr {
			if strings.HasPrefix(strings.ToLower(a.Key), "on") {
				t.Errorf("%s: payload %q produced event attribute %s:\n%s", where, payload, a.Key, out)
			}
			if (a.Key == "href" || a.Key == "src" || a.Key == "action") && strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.Val)), "javascript:") {
				t.Errorf("%s: payload %q produced a javascript: URL:\n%s", where, payload, out)
			}
		}
	})
}

// TestRenderIgnoresExtraNesting makes sure a prop value that is an object
// where a string is expected is rejected rather than stringified.
func TestRenderRejectsWrongTypes(t *testing.T) {
	reg := builtins(t)
	cases := map[string]map[string]any{
		"button":  {"label": map[string]any{"x": 1}},
		"list":    {"items": "not a list"},
		"table":   {"caption": "c", "columns": []any{1, 2}, "rows": []any{}},
		"heading": {"text": "x", "level": "2"},
	}
	for name, props := range cases {
		if _, err := reg.Render(name, props); err == nil {
			t.Errorf("%s: expected wrong-type props to be rejected: %v", name, props)
		}
	}
}
