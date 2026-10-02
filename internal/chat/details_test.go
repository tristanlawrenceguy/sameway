package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The prompt names each component and type in a line; details reads one
// whole when it is about to be used, and the prompt carries no schema.
func TestDetailsReadsWhatThePromptOnlyNames(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	if strings.Contains(m.seen[0].System, `"additionalProperties"`) {
		t.Error("the prompt carries no schema, only a line each")
	}
	for name, want := range map[string]string{"list": "Props schema:", "note": "Fields schema:", "nonesuch": "nothing is called"} {
		raw, _ := json.Marshal(map[string]any{"name": name})
		text, _ := svc.Call("details", raw)
		if !strings.Contains(text, want) {
			t.Errorf("details %s should say %q: %.200s", name, want, text)
		}
	}
	raw, _ := json.Marshal(map[string]any{"name": "list"})
	if text, _ := svc.Call("details", raw); !strings.Contains(text, "Example props:") {
		t.Errorf("a component comes with an example: %.300s", text)
	}
}
