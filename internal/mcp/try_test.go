package mcp_test

import (
	"strings"
	"testing"
)

// try runs a tool on a copy: it answers as the tool would and the
// workspace is as it was; what reaches outside is not tried.
func TestTryAnswersAsTheToolWouldAndChangesNothing(t *testing.T) {
	t.Parallel()
	a, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"try","arguments":{"name":"create_record","arguments":{"type":"note","fields":{"title":"Only tried"}}}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"try","arguments":{"name":"create_record","arguments":{"type":"note","fields":{"colour":"red"}}}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"try","arguments":{"name":"run_action","arguments":{"id":"x"}}}}`,
	)
	if body, isErr := text(t, replies[0]); isErr || !strings.Contains(body, "nothing was changed") || !strings.Contains(body, "/t/note/") {
		t.Errorf("a change that would go through says what it would do: %v %s", isErr, body)
	}
	if body, isErr := text(t, replies[1]); !isErr || !strings.Contains(body, "would be refused") || !strings.Contains(body, "colour") {
		t.Errorf("a change that would be refused says so, in the tool's words: %v %s", isErr, body)
	}
	if body, isErr := text(t, replies[2]); !isErr || !strings.Contains(body, "outside") {
		t.Errorf("what reaches outside is not tried: %v %s", isErr, body)
	}
	if n, _ := a.Store.Count("note"); n != 0 {
		t.Errorf("trying made %d notes", n)
	}
}
