package mcp_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Over stdio, with no page looked at yet, a block is still checked when
// written the way its page will resolve it: a board of a type with no
// pick-list is refused as an error that says why and what to do, and
// nothing is added; one that can be shown says what it shows, and one
// that shows nothing says so loudly. Props that fit but could not mean
// what was written (a chart by a date with no period, one field asked
// for two values) are refused too.
func TestABlockWrittenOverMCPIsCheckedAsItsPageWouldBe(t *testing.T) {
	t.Parallel()
	a, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"collection","props":{"type":"entry","as":"board"}}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"calendar","props":{"type":"tasks"}}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"collection","props":{"type":"task","where":["done=false"],"order":"due"}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"chart","props":{"type":"task","by":"due"}}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"collection","props":{"type":"task","where":["status=doing","status=todo"]}}}}`,
	)
	said, isErr := text(t, replies[0])
	if !isErr || !strings.Contains(said, "a board needs a pick-list field for its columns, and entry has none; show it as a list, table or cards, or add a pick-list field first") {
		t.Errorf("a board by a yes/no field is an error with its reason and fix, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[1])
	if !isErr || !strings.Contains(said, "there is no content type tasks (did you mean task?)") {
		t.Errorf("a calendar of a type there is not is an error naming the one meant, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[2])
	if isErr || !strings.Contains(said, "; it shows nothing yet: no task matches done=false (it fills in as records are added") {
		t.Errorf("a list that shows nothing yet says so, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[3])
	if !isErr || !strings.Contains(said, "grouping by a date needs a period: day, week or month") {
		t.Errorf("a chart by a date with no period is an error, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[4])
	if !isErr || !strings.Contains(said, "status=doing and status=todo can never both hold") || !strings.Contains(said, "use one value, or one block per value") {
		t.Errorf("a field asked for two values is an error with its fix, got %v %q", isErr, said)
	}
	blocks, _ := a.Store.List("block", store.ListOptions{})
	for _, b := range blocks {
		if p, _ := b.Fields["props"].(map[string]any); p["as"] == "board" || p["type"] == "tasks" || p["by"] == "due" || p["where"] != nil && len(p["where"].([]any)) == 2 {
			t.Errorf("a refused block was added: %v", b.Fields)
		}
	}
}
