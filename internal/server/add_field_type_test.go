package server_test

import (
	"strings"
	"testing"
)

// A small model asked for a priority on tasks sent add_field with to:
// task and no type. to names a ref's kind, so on a field of another kind
// it is the kind meant; with neither, the refusal says what to send.
func TestAddFieldReadsToAsTheKindWhenNoTypeIsSent(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	if msg, isErr := a.Chat.Call("add_field", []byte(`{"name":"priority","kind":"enum","values":["low","high"],"to":"task"}`)); isErr || !strings.Contains(msg, "added priority (enum) to task") {
		t.Errorf("to as the kind: %v %s", isErr, msg)
	}
	if msg, isErr := a.Chat.Call("add_field", []byte(`{"name":"effort","kind":"int"}`)); !isErr || !strings.Contains(msg, "add_field needs type") {
		t.Errorf("no type at all is refused, saying what to send: %v %s", isErr, msg)
	}
}
