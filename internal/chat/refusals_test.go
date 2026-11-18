package chat_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// A refusal says each thing once, in the assistant's own tools, and no
// more than it takes to fix: a field's problem once and the type's fields
// once; a ref's id from find_records, not an address of the API; a
// misspelt prop its right name, without the component's whole schema.
func TestARefusalSaysWhatToFixOnce(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	call := func(name, args string) string {
		t.Helper()
		text, isErr := svc.Call(name, json.RawMessage(args))
		if !isErr {
			t.Fatalf("%s %s was taken: %s", name, args, text)
		}
		return text
	}
	if text := call("create_record", `{"type":"task","fields":{"title":"x","status":"later"}}`); strings.Count(text, "todo, doing, done") != 2 || strings.Count(text, "takes, inside fields") != 1 {
		t.Errorf("the problem once, the fields once: %s", text)
	}
	if text := call("create_record", `{"type":"task","fields":{"title":"Pay rent","priority":"high"}}`); !strings.Contains(text, "add_field first") {
		t.Errorf("a field the type lacks says how to have one: %s", text)
	}
	if text := call("create_record", `{"type":"entry","fields":{"amount":2}}`); strings.Contains(text, "/api/") || !strings.Contains(text, "find_records on habit") {
		t.Errorf("a ref is found with the assistant's own tools: %s", text)
	}
	if text := call("add_component", `{"component":"collection","props":{"type":"task","wher":["done=false"]}}`); !strings.Contains(text, "takes where, not wher") || len(text) > 600 {
		t.Errorf("a slip of a letter is told its fix, briefly (%d chars): %s", len(text), text)
	}
}
