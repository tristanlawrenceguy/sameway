package render_test

import (
	"strings"
	"testing"
)

// Each change under a reply is an event, compact: the same line the log
// shows, without the actor the message's heading already says, never a
// heading of its own, and its Undo back to the page the message is on.
func TestMessageChangesAreCompactEvents(t *testing.T) {
	reg := builtins(t)
	out, err := reg.Render("message", map[string]any{
		"role": "assistant", "content": "Done.", "from": "/chat",
		"changes": []any{
			map[string]any{"action": "changed", "target": "text size", "detail": "to Large", "undo": "/activity/a1/undo"},
			map[string]any{"action": "created", "target": "note", "detail": "Seeds to buy", "href": "/t/note/n1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	receipt := got[strings.Index(got, `<ul class="sw-message__changes">`):]
	if n := strings.Count(receipt, `data-component="event"`); n != 2 {
		t.Fatalf("each change should be an event, got %d\n%s", n, got)
	}
	if strings.Count(receipt, "sw-event--compact") != 2 || strings.Contains(receipt, "sw-event__actor") || strings.Contains(receipt, "sw-event__mark") {
		t.Errorf("a change under a reply leaves the actor to the message\n%s", receipt)
	}
	if strings.Contains(receipt, "<h") {
		t.Errorf("a change under a reply is not a heading\n%s", receipt)
	}
	if !strings.Contains(receipt, `<form class="sw-event__undo" method="post" action="/activity/a1/undo"><input type="hidden" name="from" value="/chat">`) {
		t.Errorf("Undo posts to the entry and returns to the message's page\n%s", receipt)
	}
	if !strings.Contains(receipt, `<a class="sw-event__link" href="/t/note/n1"><span class="sw-event__target">note</span> <span class="sw-event__detail">Seeds to buy</span></a>`) {
		t.Errorf("a change leads to what it made\n%s", receipt)
	}
	for _, gone := range []string{"sw-message__change-", "sw-message__undo"} {
		if strings.Contains(got, gone) {
			t.Errorf("the message's own change markup is gone, found %s", gone)
		}
	}
}

// Undo button accessible name for undo entries omits the action word "undid"
// or "put back", so screen readers announce "Undo Assistant changed pace to
// Calmly" instead of "Undo undid Assistant changed pace to Calmly". This
// covers acceptance item 2 of task 0228.
func TestEventUndoButtonAccessibleName(t *testing.T) {
	reg := builtins(t)

	// When action is "undid", the button's accessible name should not repeat it.
	out, err := reg.Render("event", map[string]any{
		"actor":  "assistant",
		"action": "undid",
		"detail": "Assistant changed pace to Calmly",
		"time":   "14:05",
		"undo":   "/activity/a1/undo",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	// The visible button text is just "Undo".
	if !strings.Contains(got, `<button type="submit" class="sw-button sw-button--quiet sw-pressable">Undo`) {
		t.Errorf("expected Undo button\n%s", got)
	}

	// The accessible name should not contain the action word for undo verbs.
	visuallyHidden := got[strings.Index(got, `sw-visually-hidden`):]
	if strings.Contains(visuallyHidden, "undid") {
		t.Errorf("undo button accessible name repeats 'undid': %s", visuallyHidden)
	}

	// The detail should still be present in the accessible name.
	if !strings.Contains(visuallyHidden, "Assistant changed pace to Calmly") {
		t.Errorf("button accessible name should include the detail: %s", visuallyHidden)
	}
}

func TestEventPutBackButtonAccessibleName(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("event", map[string]any{
		"actor":  "human",
		"action": "put back",
		"detail": "Assistant changed text size to Large",
		"time":   "15:30",
		"undo":   "/activity/a2/undo",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	visuallyHidden := got[strings.Index(got, `sw-visually-hidden`):]
	if strings.Contains(visuallyHidden, "put back") {
		t.Errorf("undo button accessible name repeats 'put back': %s", visuallyHidden)
	}
	if !strings.Contains(visuallyHidden, "Assistant changed text size to Large") {
		t.Errorf("button accessible name should include the detail: %s", visuallyHidden)
	}
}

// A message refuses a change in the stored shape: the server says it in
// words first, so a key never reaches the page.
func TestMessageChangesTakeEventProps(t *testing.T) {
	reg := builtins(t)
	if _, err := reg.Render("message", map[string]any{
		"role": "assistant", "content": "Done.",
		"changes": []any{map[string]any{"action": "set", "component": "ui.text", "detail": "large"}},
	}); err == nil {
		t.Error("a change with a component key should be refused")
	}
}
