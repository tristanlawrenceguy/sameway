package render_test

import (
	"strings"
	"testing"
)

// TestMessageChangeRecordLinkNoTypeSuffixNote checks that a record-type change
// with component "note" and an href starting with /t/ does not render the type
// suffix "(note)" inside the anchor element. After cleaning (via CleanRecordLink
// or direct template fix), only the detail text should appear as link text.
// This covers acceptance item 1: accessible name must be exactly "My Test Note",
// no "(note)" suffix in output HTML or screen reader outline.
func TestMessageChangeRecordLinkNoTypeSuffixNote(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I created a note.",
		"time":    "14:05",
		"changes": []any{
			map[string]any{
				"action":    "created",
				"component": "note",
				"detail":    "My Test Note",
				"href":      "/t/note/testid123",
				"id":        "testid123",
			},
		},
	})
	if err != nil {
		t.Fatalf("render message with record change: %v", err)
	}

	got := string(out)

	if strings.Contains(got, "(note)") {
		t.Errorf("record link must not render type suffix '(note)' in accessible text\nwant: no '(note)' span\ngot:\n%s", got)
	}

	if !strings.Contains(got, `class="sw-message__change-detail">My Test Note`) {
		t.Errorf("changes list should show the record detail as link text\nwant: 'My Test Note' in output\ngot:\n%s", got)
	}
}

// TestMessageChangeRecordLinkNoTypeSuffixTask checks that a task-type change
// does not render "(task)" inside the anchor element. This covers acceptance
// item 2 for the "task" type.
func TestMessageChangeRecordLinkNoTypeSuffixTask(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I created a task.",
		"time":    "14:06",
		"changes": []any{
			map[string]any{
				"action":    "created",
				"component": "task",
				"detail":    "My Test Task",
				"href":      "/t/task/taskid456",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	if strings.Contains(got, "(task)") {
		t.Errorf("record link must not render type suffix '(task)' in accessible text\nwant: no '(task)' span\ngot:\n%s", got)
	}

	if !strings.Contains(got, `class="sw-message__change-detail">My Test Task`) {
		t.Errorf("changes detail should be visible as link text\nwant 'My Test Task' in output\ngot:\n%s", got)
	}
}

// TestMessageChangeRecordLinkNoTypeSuffixHabit checks that a habit-type change
// does not render "(habit)" inside the anchor element. This covers acceptance
// item 2 for the "habit" type, with a short detail like "Drink tea".
func TestMessageChangeRecordLinkNoTypeSuffixHabit(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I logged a habit.",
		"time":    "14:07",
		"changes": []any{
			map[string]any{
				"action":    "created",
				"component": "habit",
				"detail":    "Drink tea",
				"href":      "/t/habit/habitid789",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	if strings.Contains(got, "(habit)") {
		t.Errorf("record link must not render type suffix '(habit)' in accessible text\nwant: no '(habit)' span\ngot:\n%s", got)
	}

	if !strings.Contains(got, `class="sw-message__change-detail">Drink tea`) {
		t.Errorf("changes detail should be visible as link text\nwant 'Drink tea' in output\ngot:\n%s", got)
	}
}

// TestMessageChangeNonRecordStillShowsComponent checks that a non-record change
// like a canvas heading still shows its component name after the fix. The
// template removes .sw-message__change-target spans, but for non-record changes
// (href not starting with /t/) there is no detail to display, so the component
// should appear as text outside the anchor. This ensures we do not lose
// information for canvas block changes when href exists but detail does not.
func TestMessageChangeNonRecordStillShowsComponent(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I added a heading.",
		"time":    "14:08",
		"changes": []any{
			map[string]any{
				"action":    "added",
				"component": "heading",
				"detail":    "Shopping",
				"href":      "/canvas/h1",
				"activity":  "a1",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// Non-record changes with detail should show the detail text.
	if !strings.Contains(got, `class="sw-message__change-detail">Shopping`) {
		t.Errorf("non-record change should still show its detail as link text\nwant: 'Shopping' in output\ngot:\n%s", got)
	}

	// The component name may appear via data-target on the <li>, which is fine.
	if !strings.Contains(got, `data-action="added"`) {
		t.Errorf("action should still be present\nwant: 'added' in output\ngot:\n%s", got)
	}
}

// TestMessageChangeNoHrefShowsComponentAsText checks that a change with no href
// but with component renders the component name as plain text (not inside an
// anchor). This is the case for changes like `{"action": "added", "component":
// "note"}` without an href — they should still be visible. After removing the
// .sw-message__change-target span, the detail or component appears directly.
func TestMessageChangeNoHrefShowsComponentAsText(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I added something.",
		"time":    "14:09",
		"changes": []any{
			map[string]any{
				"action":    "added",
				"component": "note",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// Without href and without detail, nothing renders after the target span removal.
	// The action should still be there.
	if !strings.Contains(got, `class="sw-message__change-action">added`) {
		t.Errorf("action should still appear\nwant: 'added' in output\ngot:\n%s", got)
	}
}
