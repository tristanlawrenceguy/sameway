package chat_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An action is a button that does something: a webhook the person set up
// is called with what they wrote, its answer is logged, and with show set
// it lives on the canvas as one block that stays current.
func TestAWebhookActionCallsOutAndShowsItsAnswer(t *testing.T) {
	var got struct {
		method, body, ct string
		calls            int
	}
	answers := []string{"18°C and clear", "12°C and raining"}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.method, got.body, got.ct = r.Method, string(raw), r.Header.Get("Content-Type")
		io.WriteString(w, answers[got.calls%2])
		got.calls++
	}))
	defer remote.Close()
	chat.HTTPClient = remote.Client()

	svc := newFullService(t)
	weather, err := svc.Store.Create(chat.ActionType, map[string]any{
		"title": "Update the weather", "kind": "webhook", "url": remote.URL + "/weather",
		"body": `{"city": "Bristol"}`, "show": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	text, isErr := press(svc, weather.ID)
	if isErr || !strings.Contains(text, "answered 200") || !strings.Contains(text, "18°C") {
		t.Fatalf("running the action should report the answer, got err=%v %q", isErr, text)
	}
	if got.method != "POST" || got.body != `{"city": "Bristol"}` || got.ct != "application/json" {
		t.Errorf("the webhook should get the body as JSON, got %+v", got)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 1 || blocks[0].Fields["props"].(map[string]any)["content"] != "18°C and clear" {
		t.Fatalf("show should put the answer on the canvas, got %v", blocks)
	}
	press(svc, weather.ID)
	blocks, _ = svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 1 || blocks[0].Fields["props"].(map[string]any)["content"] != "12°C and raining" {
		t.Errorf("a second run should update the same block, got %v", blocks)
	}
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at"})
	var ran int
	for _, e := range log {
		if e.Fields["action"] == "ran" {
			ran++
		}
	}
	if ran != 2 {
		t.Errorf("each run is in the log, got %d", ran)
	}

	// An arrangement action lays out a page; a message action is the
	// person's to press, not the assistant's, so its text goes as they wrote it.
	week, _ := svc.Store.Create(chat.ActionType, map[string]any{"title": "Plan the week", "kind": "arrangement", "arrangement": "week"})
	if text, isErr := svc.Call("run_action", json.RawMessage(`{"id":"`+week.ID+`"}`)); isErr || !strings.Contains(text, "4 blocks") {
		t.Errorf("an arrangement action should lay out the page, got err=%v %q", isErr, text)
	}
	if _, isErr := svc.Call("run_action", json.RawMessage(`{"id":"nope"}`)); !isErr {
		t.Error("an unknown action should be refused")
	}
	if _, _, err := svc.RunAs(context.Background(), "human", weather.ID, ""); err != nil {
		t.Errorf("a person can run it too: %v", err)
	}
}
