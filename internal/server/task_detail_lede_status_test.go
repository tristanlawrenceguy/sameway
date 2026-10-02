package server_test

// Task detail pages must not show raw status enum badges ("Doing", "Done") in
// the lede paragraph — backlog 0552, task 0245. The checkbox (the mark) conveys
// completion; the definition list below carries status details without exposing
// internal names. Only human-readable metadata (due date, creation time) should
// appear in the lede.

import (
	"strings"
	"testing"
)

// TestTaskDetailLedeHasNoStatusBadgeDoing asserts that an undone task whose
// status is "doing" does not show a raw "Doing" badge chip in its lede text
// under the heading (backlog 0552, acceptance item 1 & 2). The checkbox must
// remain unchecked while no status word appears in the paragraph.
func TestTaskDetailLedeHasNoStatusBadgeDoing(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title":  "Call mom",
		"due":    "2026-10-02T00:00:00Z",
		"status": "doing",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	// The lede must not contain a badge chip for the status value "Doing".
	ledeStart := strings.Index(page, `<p class="sw-lede">`)
	if ledeStart == -1 {
		t.Fatal("no lede found in task detail page")
	}
	ledeEnd := strings.Index(page[ledeStart:], `</p>`)
	if ledeEnd == -1 {
		t.Fatal("no closing tag for lede")
	}
	lede := page[ledeStart : ledeStart+ledeEnd]

	if strings.Contains(lede, ">Doing</span>") {
		t.Errorf("task detail lede must not show raw \"Doing\" status badge:\n%s", lede)
	}
	if strings.Contains(lede, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("task detail lede must not expose internal field name \"status\":\n%s", lede)
	}

	// The checkbox should be unchecked.
	if !strings.Contains(page, `name="prop-done" value="true"`) || strings.Contains(page, `name="prop-done" value="true" checked`) {
		t.Errorf("the undone task's Done checkbox must remain unchecked:\n%s", truncate(page))
	}

	// Human-readable metadata must still appear: due date chip and creation time.
	if !strings.Contains(lede, "Added ") {
		t.Error("task detail lede should still show when the record was created")
	}
}

// TestTaskDetailLedeHasNoStatusBadgeDone asserts that a task whose status is
// explicitly set to "done" (and done=true) does not show a redundant raw
// "Done" badge chip in its lede text — the checkbox already conveys this
// visually (backlog 0552, acceptance item 1). The word "Done" should appear
// only on the checkbox label.
func TestTaskDetailLedeHasNoStatusBadgeDone(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title":  "Water the plants",
		"done":   true,
		"status": "done",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	// The lede must not contain a badge chip for the status value "Done".
	ledeStart := strings.Index(page, `<p class="sw-lede">`)
	if ledeStart == -1 {
		t.Fatal("no lede found in task detail page")
	}
	ledeEnd := strings.Index(page[ledeStart:], `</p>`)
	if ledeEnd == -1 {
		t.Fatal("no closing tag for lede")
	}
	lede := page[ledeStart : ledeStart+ledeEnd]

	if strings.Contains(lede, ">Done</span>") {
		t.Errorf("task detail lede must not show raw \"Done\" status badge — checkbox already conveys completion:\n%s", lede)
	}
	if strings.Contains(lede, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("task detail lede must not expose internal field name \"status\":\n%s", lede)
	}

	// The checkbox should be checked.
	if !strings.Contains(page, `name="prop-done" value="true" checked`) {
		t.Errorf("the completed task's Done checkbox must be checked:\n%s", truncate(page))
	}

	// Human-readable metadata must still appear: creation time.
	if !strings.Contains(lede, "Added ") {
		t.Error("task detail lede should still show when the record was created")
	}
}

// TestTaskDetailLedeHasNoStatusBadgeForUndoneTask asserts that a plain task
// with no explicit status (defaulting to "To do" via its tick) does not show
// any status value like "Active", "Pending", or "To do" in the lede text
// (backlog 0552, acceptance item 2).
func TestTaskDetailLedeHasNoStatusBadgeForUndoneTask(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy groceries",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	// The lede must not contain any status badge chip.
	ledeStart := strings.Index(page, `<p class="sw-lede">`)
	if ledeStart == -1 {
		t.Fatal("no lede found in task detail page")
	}
	ledeEnd := strings.Index(page[ledeStart:], `</p>`)
	if ledeEnd == -1 {
		t.Fatal("no closing tag for lede")
	}
	lede := page[ledeStart : ledeStart+ledeEnd]

	for _, word := range []string{">To do</span>", ">Doing</span>", ">Done</span>"} {
		if strings.Contains(lede, word) {
			t.Errorf("task detail lede must not show status badge %q:\n%s", word, lede)
		}
	}
	if strings.Contains(lede, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("task detail lede must not expose internal field name \"status\":\n%s", lede)
	}

	// Human-readable metadata must still appear: creation time.
	if !strings.Contains(page, "Added ") {
		t.Error("task detail page should still show when the record was created")
	}

	// The Done checkbox must be unchecked.
	if strings.Contains(page, `name="prop-done" value="true" checked`) {
		t.Errorf("the undone task's Done checkbox must remain unchecked:\n%s", truncate(page))
	}
}
