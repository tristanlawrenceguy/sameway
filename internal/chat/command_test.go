package chat_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestHelperEcho is the program a command action runs in these tests,
// when SAMEWAY_FAKE_ECHO is set: it prints its arguments and exits with
// the code SAMEWAY_FAKE_EXIT names.
func TestHelperEcho(t *testing.T) {
	if os.Getenv("SAMEWAY_FAKE_ECHO") == "" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	fmt.Print(strings.Join(args, " "))
	code := 0
	fmt.Sscanf(os.Getenv("SAMEWAY_FAKE_EXIT"), "%d", &code)
	os.Exit(code)
}

// echoLine is a command line that runs the helper with these words.
func echoLine(words string) string {
	return `"` + os.Args[0] + `" -test.run=TestHelperEcho -- ` + words
}

func allowHelper(svc *chat.Service) {
	svc.Allow = []string{filepath.Base(os.Args[0])}
}

// A command action is written by the assistant and accepted once by the
// person, with the line in front of them; until then nothing runs. After
// that it is a plain button, and changing the line asks again.
func TestACommandRunsOnlyOnceThePersonHasAcceptedIt(t *testing.T) {
	os.Setenv("SAMEWAY_FAKE_ECHO", "1")
	defer os.Unsetenv("SAMEWAY_FAKE_ECHO")
	svc := newFullService(t)
	allowHelper(svc)
	line := echoLine("hello there")
	action, err := svc.Store.Create(chat.ActionType, map[string]any{"title": "Say hello", "kind": "command", "command": line, "show": true})
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
	if len(pending) != 1 || !strings.Contains(pending[0].Fields["summary"].(string), "hello there") {
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
	if rec.Fields["accepted"] != line {
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
	svc.Store.Update(chat.ActionType, action.ID, map[string]any{"command": echoLine("something else")})
	if text, isErr := svc.Call("run_action", raw); isErr || !strings.Contains(text, "has not been accepted") {
		t.Errorf("a changed command asks again, got err=%v %q", isErr, text)
	}
	// A failing command says so and is an error to the caller.
	os.Setenv("SAMEWAY_FAKE_EXIT", "3")
	defer os.Unsetenv("SAMEWAY_FAKE_EXIT")
	bad, _ := svc.Store.Create(chat.ActionType, map[string]any{"title": "Fail", "kind": "command", "command": echoLine("boom"), "accepted": echoLine("boom")})
	if text, isErr := svc.Call("run_action", json.RawMessage(`{"id":"`+bad.ID+`"}`)); !isErr || !strings.Contains(text, "exit code 3") {
		t.Errorf("a failing command reports its exit code as an error, got err=%v %q", isErr, text)
	}
}

// The boundary: a command runs only a program the workspace has allowed
// by name, never through a shell, and only inside the workspace folder.
func TestACommandStaysInsideTheBoundary(t *testing.T) {
	os.Setenv("SAMEWAY_FAKE_ECHO", "1")
	defer os.Unsetenv("SAMEWAY_FAKE_ECHO")
	svc := newFullService(t)

	// Not allowed: nothing runs, accepted or not, and the message says how
	// to allow it.
	action, _ := svc.Store.Create(chat.ActionType, map[string]any{"title": "Say hello", "kind": "command", "command": echoLine("hi"), "accepted": echoLine("hi")})
	raw, _ := json.Marshal(map[string]any{"id": action.ID})
	if text, isErr := svc.Call("run_action", raw); !isErr || !strings.Contains(text, "actions.allow") {
		t.Errorf("a program not on the allow list is refused with the way to allow it, got err=%v %q", isErr, text)
	}
	if len(svc.Proposals()) != 0 {
		t.Error("a refused program is not even offered for acceptance")
	}

	// Allowed by name, however the program is written.
	svc.Allow = []string{strings.ToUpper(strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe"))}
	if text, isErr := svc.Call("run_action", raw); isErr || !strings.Contains(text, "exit code 0") {
		t.Errorf("an allowed program runs, got err=%v %q", isErr, text)
	}

	// No shell: what would be a pipe or a second command is only words
	// handed to the program.
	shelly := echoLine(`one "two words" | rm -rf ; echo three && four`)
	svc.Store.Update(chat.ActionType, action.ID, map[string]any{"command": shelly, "accepted": shelly})
	text, isErr := svc.Call("run_action", raw)
	if isErr || !strings.Contains(text, "one two words | rm -rf ; echo three && four") {
		t.Errorf("the line is arguments to the program and nothing more, got err=%v %q", isErr, text)
	}

	// The folder must be the workspace or under it.
	chat.Workdir = t.TempDir()
	defer func() { chat.Workdir = "" }()
	svc.Store.Update(chat.ActionType, action.ID, map[string]any{"folder": filepath.Dir(chat.Workdir)})
	if text, isErr := svc.Call("run_action", raw); !isErr || !strings.Contains(text, "outside the workspace") {
		t.Errorf("a folder outside the workspace is refused, got err=%v %q", isErr, text)
	}
	os.MkdirAll(filepath.Join(chat.Workdir, "sub"), 0o755)
	svc.Store.Update(chat.ActionType, action.ID, map[string]any{"folder": "sub"})
	if text, isErr := svc.Call("run_action", raw); isErr || !strings.Contains(text, "exit code 0") {
		t.Errorf("a folder under the workspace is fine, got err=%v %q", isErr, text)
	}
}
