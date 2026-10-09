package chat_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// Props that do not fit are said for the model that has to fix them: the
// prop that is wrong, where a block field belongs, the prop likely meant,
// what is allowed, and every prop the component takes. The cases are the
// crew's own: tone inside props ~350 times in its logs, a button's text
// for its label ~40, an alert's kind outside its choices.
func TestPropsThatDoNotFitAreSaidForTheModelFixingThem(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	for _, c := range []struct {
		name string
		args map[string]any
		want []string
	}{
		{"tone inside props", map[string]any{"component": "text", "props": map[string]any{"content": "Water the tomatoes", "tone": "info"}},
			[]string{"tone is a block field, not a text prop: pass it beside props, not inside them", "text takes: content (required), id, level, muted."}},
		{"a button's text for its label", map[string]any{"component": "button", "props": map[string]any{"text": "Save"}},
			[]string{"text is not a button prop; button takes label, not text", "label is required and missing", "label (required)"}},
		{"a heading's label for its text", map[string]any{"component": "heading", "props": map[string]any{"label": "This week"}},
			[]string{"label is not a heading prop; heading takes text, not label", "text is required and missing"}},
		{"an alert's kind outside its choices", map[string]any{"component": "alert", "props": map[string]any{"message": "Saved", "kind": "error"}},
			[]string{`kind must be one of "info", "success", "warning", "danger", not "error"`}},
		{"a typo", map[string]any{"component": "text", "props": map[string]any{"contnet": "Hi"}},
			[]string{"contnet is not a text prop; text takes content, not contnet", "content is required and missing"}},
		{"a number as words", map[string]any{"component": "heading", "props": map[string]any{"text": "Hi", "level": "two"}},
			[]string{"level must be integer, not a string"}},
	} {
		raw, _ := json.Marshal(c.args)
		said, isErr := svc.Call("add_component", raw)
		if !isErr {
			t.Errorf("%s: expected an error, got %q", c.name, said)
			continue
		}
		for _, want := range c.want {
			if !strings.Contains(said, want) {
				t.Errorf("%s: the error should say %q:\n%s", c.name, want, said)
			}
		}
		if !strings.HasSuffix(said, "Fix the props and call add_component again.") {
			t.Errorf("%s: the error should say what to call again:\n%s", c.name, said)
		}
		for _, raw := range []string{"something I don't recognise", "additional properties", "/:"} {
			if strings.Contains(said, raw) {
				t.Errorf("%s: %q says nothing a model can act on:\n%s", c.name, raw, said)
			}
		}
	}

	// update_component is told the same way.
	raw, _ := json.Marshal(map[string]any{"component": "text", "props": map[string]any{"content": "Hi"}})
	said, isErr := svc.Call("add_component", raw)
	if isErr {
		t.Fatal(said)
	}
	id := strings.Fields(strings.SplitAfter(said, "as block ")[1])[0]
	raw, _ = json.Marshal(map[string]any{"id": id, "props": map[string]any{"content": "Hi", "span": 6}})
	said, _ = svc.Call("update_component", raw)
	if !strings.Contains(said, "span is a block field, not a text prop: pass it beside props, not inside them") || !strings.HasSuffix(said, "call update_component again.") {
		t.Errorf("update_component should say where span belongs:\n%s", said)
	}
}
