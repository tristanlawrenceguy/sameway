package mcp_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// clientLimit is what an MCP client takes from one tool call: Claude Code
// refuses more than 25,000 tokens, and 17 of 25 describe calls in the
// evaluation were refused. A token is about four bytes of this JSON; three
// is counted, to be safe.
const clientLimit = 25000 * 3

// Nothing describe answers unless asked for everything is more than a
// client takes: not the index, not a part, not one thing by name. And a
// name alone finds the thing, where before it was taken for no argument
// and answered with all of it.
func TestDescribeOverMCPFitsTheClient(t *testing.T) {
	calls := []string{`{}`, `{"name":"meter"}`, `{"name":"task"}`, `{"name":"block_add"}`, `{"part":"index"}`}
	for _, p := range app.Parts {
		if p != "full" {
			calls = append(calls, fmt.Sprintf(`{"part":%q}`, p))
		}
	}
	var lines []string
	for i, args := range calls {
		lines = append(lines, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"describe","arguments":%s}}`, i+1, args))
	}
	_, replies := drive(t, lines...)
	if len(replies) != len(calls) {
		t.Fatalf("%d calls, %d replies", len(calls), len(replies))
	}
	for i, args := range calls {
		body, isErr := text(t, replies[i])
		t.Logf("describe %s: %d bytes", args, len(body))
		if isErr {
			t.Errorf("describe %s failed: %.200s", args, body)
		}
		if len(body) > clientLimit {
			t.Errorf("describe %s is %d bytes, more than a client takes (%d)", args, len(body), clientLimit)
		}
	}
	index, _ := text(t, replies[0])
	if len(index) > 16*1024 || !strings.Contains(index, "POST /api/block") {
		t.Errorf("describe with nothing asked should be the index, short, saying how to add a block: %d bytes", len(index))
	}
	meter, _ := text(t, replies[1])
	for _, want := range []string{`"name": "meter"`, `"example"`, `"props"`} {
		if !strings.Contains(meter, want) {
			t.Errorf("describe name meter should be the meter's props with an example, missing %s: %.300s", want, meter)
		}
	}
	if strings.Contains(meter, `"a11y"`) || strings.Contains(meter, `"machine"`) {
		t.Error("describe name meter should be compact, without the accessibility and machine contract")
	}
	if task, _ := text(t, replies[2]); !strings.Contains(task, `"due"`) {
		t.Errorf("describe name task should be the task type: %.200s", task)
	}
}
