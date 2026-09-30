package chat_test

import (
	"strings"
	"testing"
)

// A model that sends a number as a string is told so in its own terms.
// The evaluation's MCP agent was answered "json: cannot unmarshal string
// into Go struct field .position of type int", Go's words, not its own.
func TestArgumentOfTheWrongKindIsSaidPlainly(t *testing.T) {
	svc, _ := newService(t)
	msg, isErr := svc.Call("add_component", []byte(`{"component": "text", "props": {"text": "Hi"}, "position": "3"}`))
	if !isErr || !strings.Contains(msg, "position is a number; you sent a string") {
		t.Errorf("a position sent as a string should be said plainly, got %q", msg)
	}
	for _, bad := range []string{"unmarshal", "Go struct", "int"} {
		if strings.Contains(msg, bad) {
			t.Errorf("the error should not carry the decoder's %q: %q", bad, msg)
		}
	}
	msg, _ = svc.Call("add_component", []byte(`{"component": "text", "span": 4.5}`))
	if !strings.Contains(msg, "span is a whole number; you sent 4.5") {
		t.Errorf("a fraction for a whole number should be said plainly, got %q", msg)
	}
	msg, _ = svc.Call("add_component", []byte(`{"component": `))
	if !strings.Contains(msg, "not valid JSON") {
		t.Errorf("broken JSON should say so, got %q", msg)
	}
}
