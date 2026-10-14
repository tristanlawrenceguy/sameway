package mcp_test

import (
	"strings"
	"testing"
)

// MCP's own describe refuses an argument it does not take, in the words
// the assistant's tools use, instead of answering with the whole index.
func TestMCPsOwnToolsRefuseAnArgumentTheyDoNotTake(t *testing.T) {
	t.Parallel()
	_, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"describe","arguments":{"nme":"task"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"describe","arguments":{"name":"task"}}}`,
	)
	if said, isErr := text(t, replies[0]); !isErr || !strings.Contains(said, "describe does not take nme (did you mean name?). It takes name, part") {
		t.Errorf("nme is refused with the one meant, got %v %q", isErr, said)
	}
	if said, isErr := text(t, replies[1]); isErr {
		t.Errorf("name still reads one thing: %q", said)
	}
}
