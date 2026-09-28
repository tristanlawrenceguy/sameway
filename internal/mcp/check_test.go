package mcp_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Over stdio, with no page looked at yet, a block is still checked when
// written the way its page will resolve it: a board of a type with no
// pick-list is refused as an error that says why and what to do, and
// nothing is added; one that can be shown says what it shows.
func TestABlockWrittenOverMCPIsCheckedAsItsPageWouldBe(t *testing.T) {
	a, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"collection","props":{"type":"task","as":"board","by":"done"}}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"calendar","props":{"type":"tasks"}}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"collection","props":{"type":"task","where":["done=false"],"order":"due"}}}}`,
	)
	said, isErr := text(t, replies[0])
	if !isErr || !strings.Contains(said, "a board needs a pick-list field for its columns, and task has none; show it as a list, table or cards, or add a pick-list field first") {
		t.Errorf("a board by a yes/no field is an error with its reason and fix, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[1])
	if !isErr || !strings.Contains(said, "there is no content type tasks (did you mean task?)") {
		t.Errorf("a calendar of a type there is not is an error naming the one meant, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[2])
	if isErr || !strings.Contains(said, "; it shows 0 tasks, not done, by due") {
		t.Errorf("a list that can be shown says what it shows, got %v %q", isErr, said)
	}
	blocks, _ := a.Store.List("block", store.ListOptions{})
	for _, b := range blocks {
		if p, _ := b.Fields["props"].(map[string]any); p["as"] == "board" || p["type"] == "tasks" {
			t.Errorf("a refused block was added: %v", b.Fields)
		}
	}
}
