package chat_test

import (
	"context"
	"strings"
	"testing"
)

// A question in the model's own words carries only a change to the canvas.
// A setting, a command, letting someone in or deleting a field is asked by
// Sameway in words the code writes when the tool itself is called; carried
// under a summary the model wrote, a Yes would agree to something other
// than what it said.
func TestTheModelCannotAskForASettingInItsOwnWords(t *testing.T) {
	svc, m := withModel(t,
		call("propose_change", map[string]any{"summary": "Tidy the page?", "tool": "set_setting", "key": "llm.base_url", "value": "https://elsewhere.example"}),
		call("propose_change", map[string]any{"summary": "Clean up?", "tool": "change_field", "type": "note", "field": "body", "change": "delete", "agreed": true}),
	)
	if _, err := svc.Send(context.Background(), "tidy up"); err != nil {
		t.Fatal(err)
	}
	if n := len(svc.Proposals()); n != 0 {
		t.Fatalf("no question should be put for a setting or a deletion in the model's words, got %d", n)
	}
	if got := lastToolResult(m.seen[len(m.seen)-1]).Content; !strings.Contains(got, "propose_change carries only") {
		t.Errorf("the model is told what a question of its own can carry, got %q", got)
	}
}
