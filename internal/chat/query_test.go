package chat_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant finds records with the same conditions a collection block
// takes, and a wrong condition comes back saying what the type has.
func TestFindRecordsTakesTheSameQuery(t *testing.T) {
	svc := newFullService(t)
	day := func(d int) string { return time.Now().AddDate(0, 0, d).UTC().Format(time.RFC3339) }
	svc.Store.Create("task", map[string]any{"title": "Dig the pond", "due": day(-2)})
	svc.Store.Create("task", map[string]any{"title": "Order compost", "due": day(2)})
	svc.Store.Create("task", map[string]any{"title": "Call the dentist", "due": day(3), "done": true})

	m := &scripted{steps: []*llm.Response{
		call("find_records", map[string]any{"type": "task", "where": []string{"done=false", "due>today"}, "order": "due"}),
		call("find_records", map[string]any{"type": "task", "where": []string{"owner=me"}}),
	}}
	svc.Provider = m
	svc.Send(context.Background(), "what is coming up?")
	found := lastToolResult(m.seen[1])
	if found.IsError || !strings.Contains(found.Content, "Order compost") || strings.Contains(found.Content, "Dig the pond") || strings.Contains(found.Content, "dentist") {
		t.Errorf("find_records should filter with the query: %+v", found)
	}
	if wrong := lastToolResult(m.seen[2]); !wrong.IsError || !strings.Contains(wrong.Content, `no field "owner"`) || !strings.Contains(wrong.Content, "due") {
		t.Errorf("a wrong field names the fields there are: %+v", wrong)
	}
	if system := m.seen[0].System; !strings.Contains(system, "collection block") || !strings.Contains(system, "due<=+7d") {
		t.Error("the prompt tells the model how to show what matches")
	}
}
