package chat

import (
	"encoding/json"
	"strings"
	"testing"
)

// A prop that takes the writer's shape or the server's (a tracker's
// habits: names, or each habit as it stands) is given to the model as the
// writer's alone.
func TestForModelKeepsOnlyTheWritersShape(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","properties":{"habits":{"type":"array","items":{"anyOf":[{"type":"string"},{"description":"Filled in by the server: a habit as it stands.","type":"object","properties":{"streak":{"type":"integer"}}}]}}}}`)
	got := string(ForModel(raw))
	if strings.Contains(got, "anyOf") || strings.Contains(got, "streak") || !strings.Contains(got, `"items":{"type":"string"}`) {
		t.Errorf("only the names are the model's to give, got %s", got)
	}
}
