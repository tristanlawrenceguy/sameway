package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The type index says the names on more than one open record that nothing
// tells apart, with keep_one to make them one; a name on one record, or on
// two of different days, is not said.
func TestTheTypeIndexSaysRecordsThatAreOneThingTwice(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Store.Create("task", map[string]any{"title": "Dentist appointment"})
	svc.Store.Create("task", map[string]any{"title": "Dentist appointment"})
	svc.Store.Create("task", map[string]any{"title": "Water the plants", "due": "2026-10-12"})
	svc.Store.Create("task", map[string]any{"title": "Water the plants", "due": "2026-10-19"})
	svc.Store.Create("task", map[string]any{"title": "Call mum"})
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	sys := m.seen[0].System
	line := sys[strings.Index(sys, "\ntask: "):]
	line = line[:strings.Index(line[1:], "\n")+1]
	if !strings.Contains(line, "1 name is on more than one open task with nothing to tell them apart: Dentist appointment (2)") || !strings.Contains(line, "keep_one") {
		t.Errorf("the twin is said, with keep_one:%s", line)
	}
	if strings.Contains(line, "Water the plants") || strings.Contains(line, "Call mum") {
		t.Errorf("tasks of different days, or one of a name, are not said:%s", line)
	}
}
