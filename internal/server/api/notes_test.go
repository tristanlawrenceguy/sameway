package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// What is worth saying about a change is said the same whichever way it
// came in: a task made over the JSON API and one made with the
// assistant's tool over /api/tools are each told of the open task called
// the same, in the same words, and a second block over the same records
// is told of the first. One place works it out (records/notes.go).
func TestEveryWayInSaysTheSameNotes(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	old, err := a.Store.Create("task", map[string]any{"title": "Pay rent"})
	if err != nil {
		t.Fatal(err)
	}
	var made struct {
		Notes []string `json:"notes"`
	}
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Pay rent"}).Body.Bytes(), &made)
	if len(made.Notes) != 1 || !strings.Contains(made.Notes[0], "called the same") || !strings.Contains(made.Notes[0], old.ID) {
		t.Fatalf("the JSON API says the open task called the same: %v", made.Notes)
	}
	tool := do(t, h, http.MethodPost, "/api/tools/create_record", strings.NewReader(`{"type":"task","fields":{"title":"Pay rent"}}`), "application/json").Body.String()
	if !strings.Contains(tool, strings.SplitN(made.Notes[0], ". ", 2)[1]) || strings.Count(tool, "called the same") != 1 {
		t.Errorf("the tool says it in the same words as the API (%q): %s", made.Notes[0], tool)
	}

	block := map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}}
	postJSON(t, h, http.MethodPost, "/api/block", block)
	var second struct {
		Notes []string `json:"notes"`
	}
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/block", block).Body.Bytes(), &second)
	if len(second.Notes) != 1 || !strings.HasPrefix(second.Notes[0], "This tab already had a collection of the same records") {
		t.Errorf("a second block over the same records is told of the first: %v", second.Notes)
	}
}
