package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// What the model reads as JSON (the canvas listing's props, a record from
// get_record) has <, > and & as written: a condition reads due<today.
func TestTheModelReadsConditionsAsWritten(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	rec, _ := svc.Store.Create("note", map[string]any{"title": "Tom & Jerry"})
	run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Late", "where": []string{"done=false", "due<today"}}})
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if sys := m.seen[0].System; !strings.Contains(sys, `"due<today"`) || strings.Contains(sys, `\u003c`) {
		t.Errorf("the listing says due<today as written")
	}
	if got := run(t, svc, "get_record", map[string]any{"type": "note", "id": rec.ID}); !strings.Contains(got, "Tom & Jerry") {
		t.Errorf("get_record says & as written: %s", got)
	}
}
