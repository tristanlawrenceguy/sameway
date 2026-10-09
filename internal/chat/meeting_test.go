package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const standup = "[0:00] **Ann:** Morning, two things.\n\n[0:12] **Ben:** We ship on Friday, then.\n\n[1:05] **Ann:** Ben, can you email the client?"

// A recording is written up as a meeting: one made for it, with its
// summary and decisions, and its tasks, each linked to the line where it
// was said. It is one entry in the log, and one Undo takes it all back.
func TestAMeetingIsWrittenUpFromItsRecording(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	file, err := svc.Store.Create(records.FileType, map[string]any{"title": "Monday stand-up", "kind": "audio", "text": standup})
	if err != nil {
		t.Fatal(err)
	}
	out := run(t, svc, "write_up_meeting", map[string]any{
		"recording": file.ID,
		"summary":   "Ann and Ben agreed the release date.",
		"decisions": []any{map[string]any{"text": "Ship on Friday", "at": "0:14"}},
		"tasks":     []any{map[string]any{"title": "Email the client", "at": "[1:05]"}, map[string]any{"title": "Book the room"}},
	})
	if !strings.Contains(out, "1 decision and 2 tasks") || !strings.Contains(out, "?show=points-here:task.event") {
		t.Errorf("it says what it wrote: %s", out)
	}
	events, _ := svc.Store.List(records.EventType, store.ListOptions{})
	if len(events) != 1 {
		t.Fatalf("a meeting is made for the recording, got %d", len(events))
	}
	ev := events[0].Fields
	if ev["recording"] != file.ID || ev["summary"] != "Ann and Ben agreed the release date." || ev["title"] != "Meeting: Monday stand-up" {
		t.Errorf("the meeting keeps its recording and summary: %v", ev)
	}
	// 0:14 is during the line that began at 0:12, so the link is to it.
	if d, _ := ev["decisions"].(string); d != "- Ship on Friday ([at 0:12](/t/file/"+file.ID+"#media-"+file.ID+"-at-12))" {
		t.Errorf("a decision links to the line it was said in: %q", d)
	}
	tasks, _ := svc.Store.List("task", store.ListOptions{OrderBy: "created_at"})
	if len(tasks) != 2 || tasks[0].Fields["event"] != events[0].ID || !strings.Contains(tasks[0].Fields["notes"].(string), "#media-"+file.ID+"-at-65") {
		t.Fatalf("each task came up at the meeting, linked to where: %+v", tasks)
	}
	if n, _ := tasks[1].Fields["notes"].(string); n != "" {
		t.Errorf("a task with no time says nothing of where: %q", n)
	}
	if w := svc.Writers().Of("task", tasks[0]).Words; w != "the assistant" {
		t.Errorf("the tasks were written by the assistant: %q", w)
	}

	wrote := entries(t, svc, "wrote up")
	if len(wrote) != 1 {
		t.Fatalf("a write-up is one entry, got %d", len(wrote))
	}
	run(t, svc, "undo_change", map[string]any{"id": wrote[0].ID})
	if events, _ := svc.Store.List(records.EventType, store.ListOptions{}); len(events) != 0 {
		t.Error("undo takes the meeting it made away")
	}
	if tasks, _ := svc.Store.List("task", store.ListOptions{}); len(tasks) != 0 {
		t.Error("undo takes its tasks away")
	}
}

// A meeting that is an event already is written up in place, and Undo
// puts it back as it was.
func TestAnEventIsWrittenUpInPlace(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	file, _ := svc.Store.Create(records.FileType, map[string]any{"title": "Call", "kind": "audio", "text": standup})
	ev, err := svc.Store.Create(records.EventType, map[string]any{"title": "Weekly call", "recording": file.ID})
	if err != nil {
		t.Fatal(err)
	}
	run(t, svc, "write_up_meeting", map[string]any{"event": ev.ID, "summary": "Short."})
	now, _ := svc.Store.Get(records.EventType, ev.ID)
	if now.Fields["summary"] != "Short." || now.Fields["title"] != "Weekly call" {
		t.Errorf("written up in place: %v", now.Fields)
	}
	run(t, svc, "undo_change", map[string]any{"id": entries(t, svc, "wrote up")[0].ID})
	if back, _ := svc.Store.Get(records.EventType, ev.ID); back.Fields["summary"] != "" {
		t.Errorf("undo puts it back as it was: %v", back.Fields)
	}
	refused(t, svc, "write_up_meeting", map[string]any{"summary": "Nothing to write it on."}, "give the event")
}
