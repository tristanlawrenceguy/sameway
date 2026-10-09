package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The prompt names each component and type in a line and carries no
// schema; a call that does not fit is refused with the whole of what it
// takes, so there is nothing to read first.
func TestARefusalCarriesWhatTheCallTakes(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	if strings.Contains(m.seen[0].System, `"additionalProperties"`) {
		t.Error("the prompt carries no schema, only a line each")
	}
	raw, _ := json.Marshal(map[string]any{"component": "list", "props": map[string]any{"text": "milk"}})
	text, isErr := svc.Call("add_component", raw)
	if !isErr || !strings.Contains(text, "Props schema:") || !strings.Contains(text, "Example props:") {
		t.Errorf("a component refused gives its schema and an example: %.400s", text)
	}
	raw, _ = json.Marshal(map[string]any{"type": "note", "fields": map[string]any{"body": "no title"}})
	text, isErr = svc.Call("create_record", raw)
	if !isErr || !strings.Contains(text, "note takes, inside fields: title (text, required)") {
		t.Errorf("a record refused gives the type's fields: %.400s", text)
	}
}
