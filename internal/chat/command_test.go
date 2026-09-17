package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A command action is written by the assistant and accepted once by the
// person, with the line in front of them; until then nothing runs. After
// that it is a plain button, and changing the line asks again.
func TestACommandRunsOnlyOnceThePersonHasAcceptedIt(t *testing.T) {
	svc := newFullService(t)
	action, err := svc.Store.Create(chat.ActionType, map[string]any{"title": "Say hello", "kind": "command", "command": "echo hello there", "show": true})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"id": action.ID})

	// The assistant asking to run it turns into a question, not a run.
	text, isErr := svc.Call("run_action", raw)
	if isErr || !strings.Contains(text, "has not been accepted") {
		t.Fatalf("an unaccepted command becomes a question, got err=%v %q", isErr, text)
	}
	pending := svc.Proposals()
	if len(pending) != 1 || !strings.Contains(pending[0].Fields["summary"].(string), "echo hello there") {
		t.Fatalf("the question should show the command line, got %v", pending)
	}
	if blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{}); len(blocks) != 0 {
		t.Fatal("nothing should have run before acceptance")
	}

	// Yes accepts it for good and runs it; the answer lands on the canvas.
	if err := svc.Accept(pending[0].ID); err != nil {
		t.Fatal(err)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 1 || !strings.Contains(blocks[0].Fields["props"].(map[string]any)["content"].(string), "hello there") {
		t.Fatalf("accepting should run the command and show what it printed, got %v", blocks)
	}
	rec, _ := svc.Store.Get(chat.ActionType, action.ID)
	if rec.Fields["accepted"] != "echo hello there" {
		t.Errorf("the accepted line is kept on the action, got %v", rec.Fields["accepted"])
	}

	// From now on it just runs, for the assistant and for the person.
	if text, isErr := svc.Call("run_action", raw); isErr || !strings.Contains(text, "exit code 0") {
		t.Errorf("an accepted command runs, got err=%v %q", isErr, text)
	}
	if _, proposal, err := svc.RunAs(context.Background(), "human", action.ID, ""); err != nil || proposal != "" {
		t.Errorf("a person's press runs it without asking again: %v %q", err, proposal)
	}
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{})
	ran := 0
	for _, e := range log {
		if e.Fields["action"] == "ran" {
			ran++
		}
	}
	if ran != 3 {
		t.Errorf("three runs in the log, got %d", ran)
	}

	// A changed command line is a new question.
	svc.Store.Update(chat.ActionType, action.ID, map[string]any{"command": "echo something else"})
	if text, isErr := svc.Call("run_action", raw); isErr || !strings.Contains(text, "has not been accepted") {
		t.Errorf("a changed command asks again, got err=%v %q", isErr, text)
	}
	// A failing command says so and is an error to the caller.
	bad, _ := svc.Store.Create(chat.ActionType, map[string]any{"title": "Fail", "kind": "command", "command": "exit 3", "accepted": "exit 3"})
	if text, isErr := svc.Call("run_action", json.RawMessage(`{"id":"`+bad.ID+`"}`)); !isErr || !strings.Contains(text, "exit code 3") {
		t.Errorf("a failing command reports its exit code as an error, got err=%v %q", isErr, text)
	}
}
