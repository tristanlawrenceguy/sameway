package chat_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// With SAMEWAY_CLAUDE_CODE_MODEL set (haiku will do), a real model is
// asked what the recording page's press asks, and writes the meeting up
// with the tool: a decision and a task, linked to where they were said.
func TestARealModelWritesUpAMeeting(t *testing.T) {
	t.Parallel()
	model := os.Getenv("SAMEWAY_CLAUDE_CODE_MODEL")
	if model == "" {
		t.Skip("set SAMEWAY_CLAUDE_CODE_MODEL to run one real turn through Claude Code")
	}
	svc := newFullService(t)
	c := llm.Presets["claude-code"]
	c.Model, c.Workspace, c.Timeout = model, t.TempDir(), 5*time.Minute
	svc.Provider = &c
	file, err := svc.Store.Create(records.FileType, map[string]any{"title": "Monday stand-up", "kind": "audio", "text": standup})
	if err != nil {
		t.Fatal(err)
	}
	ask := "Write up the meeting in this recording (/t/file/" + file.ID + "): a short summary, what was decided and the tasks that came up."
	if _, err := svc.Send(context.Background(), ask); err != nil {
		t.Fatal(err)
	}
	events, _ := svc.Store.List(records.EventType, store.ListOptions{})
	if len(events) != 1 {
		t.Fatalf("the model made one meeting, got %d", len(events))
	}
	d, _ := events[0].Fields["decisions"].(string)
	if !strings.Contains(strings.ToLower(d), "friday") || !strings.Contains(d, "-at-12") {
		t.Errorf("shipping on Friday is a decision, linked to 0:12: %q", d)
	}
	tasks, _ := svc.Store.List("task", store.ListOptions{})
	found := false
	for _, task := range tasks {
		notes, _ := task.Fields["notes"].(string)
		found = found || strings.Contains(strings.ToLower(task.Fields["title"].(string)), "client") && strings.Contains(notes, "-at-65")
	}
	if !found {
		t.Errorf("emailing the client is a task, linked to 1:05: %+v", tasks)
	}
}
