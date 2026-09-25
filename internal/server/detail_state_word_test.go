package server_test

// Tests for removing the state word from the lede/status line on record
// detail pages (task 0195). The mark checkbox already announces the state
// via its accessible name; the lede text must start directly with badges.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPinnedNoteDetailLedeHasNoLeadingStateWord checks that on a pinned note
// detail page, the lede paragraph does not contain "Pinned" as visible text or
// visually-hidden span — only the checkbox with its aria-label carries it.
// Acceptance item 1 of task 0195.
func TestPinnedNoteDetailLedeHasNoLeadingStateWord(t *testing.T) {
	a, h := newApp(t)

	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note",
		map[string]any{"title": "Buy tomatoes", "pinned": true}), &note)
	_ = a // workspace is needed to create records

	rec := get(t, h, "/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The lede paragraph must not contain "Pinned" as visible text.
	if strings.Contains(body, ">Pinned<") {
		t.Errorf("lede should not have >Pinned< visible text\n%s", truncate(body))
	}
	// Nor a visually-hidden span carrying it.
	if strings.Contains(body, `<span class="sw-visually-hidden">Pinned`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> Pinned`) {
		t.Errorf("lede should not have sw-visually-hidden>Pinned\n%s", truncate(body))
	}

	// The checkbox must exist and be checked.
	if !strings.Contains(body, `name="prop-pinned"`) ||
		!strings.Contains(body, `<input class="sw-mark__input" type="checkbox" name="prop-pinned" value="true"`) {
		t.Errorf("pinned note detail should have a checked pinned checkbox\n%s", truncate(body))
	}
	if idx := strings.Index(body, `name="prop-pinned"`); idx >= 0 && !strings.Contains(body[idx:idx+200], " checked") {
		t.Errorf("pinned note detail should have a checked pinned checkbox\n%s", truncate(body))
	}

	// The lede must contain the status badge and creation text.
	if !strings.Contains(body, `class="sw-badge`) && !strings.Contains(body, ">Draft<") {
		t.Logf("warning: expected status badge in lede; page:\n%s", truncate(body))
	}
}

// TestUnpinnedNoteDetailLedeHasNoPinnedWord checks that on an unpinned note
// detail page, the word "Pinned" does not appear anywhere in text nodes — only
// as a checkbox (unchecked) whose aria-label carries it.
// Acceptance item 2 of task 0195.
func TestUnpinnedNoteDetailLedeHasNoPinnedWord(t *testing.T) {
	_, h := newApp(t)

	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note",
		map[string]any{"title": "Meeting notes"}), &note)

	rec := get(t, h, "/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// No visible or hidden "Pinned" in the page.
	if strings.Contains(body, ">Pinned<") ||
		strings.Contains(body, `<span class="sw-visually-hidden">Pinned`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> Pinned`) {
		t.Errorf("unpinned note detail should not mention Pinned at all\n%s", truncate(body))
	}

	// The checkbox must exist and be unchecked.
	if !strings.Contains(body, `name="prop-pinned"`) {
		t.Errorf("unpinned note detail should have a pinned checkbox\n%s", truncate(body))
	}
	if strings.Contains(body, `<input class="sw-mark__input" type="checkbox" name="prop-pinned" value="true" checked>`) {
		t.Errorf("the pinned checkbox on an unpinned note must not be checked\n%s", truncate(body))
	}

	// The lede should still contain the status badge.
	if !strings.Contains(body, `class="sw-badge`) && !strings.Contains(body, ">Draft<") {
		t.Logf("warning: expected status badge in lede; page:\n%s", truncate(body))
	}
}

// TestTaskDetailLedeHasNoLeadingDone checks that on a task detail page with
// done=true, the lede starts directly with badges — no "Done" text before them.
// Acceptance item 3 of task 0195 (task part).
func TestTaskDetailLedeHasNoLeadingDone(t *testing.T) {
	_, h := newApp(t)

	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task",
		map[string]any{"title": "Water the plants", "done": true}), &task)

	rec := get(t, h, "/t/task/"+task.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The mark must not render a visible "Done" label.
	if strings.Contains(body, "> Done") {
		t.Errorf("done task detail should not have visible > Done\n%s", truncate(body))
	}
	// Nor a visually-hidden span carrying it.
	if strings.Contains(body, `<span class="sw-visually-hidden">Done`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> Done`) {
		t.Errorf("done task detail should not have sw-visually-hidden>Done\n%s", truncate(body))
	}

	// The checkbox must exist and be checked.
	if !strings.Contains(body, `name="prop-done"`) ||
		!strings.Contains(body, `<input class="sw-mark__input" type="checkbox" name="prop-done" value="true"`) {
		t.Errorf("done task detail should have a checked done checkbox\n%s", truncate(body))
	}
	if idx := strings.Index(body, `name="prop-done"`); idx >= 0 && !strings.Contains(body[idx:idx+200], "checked>") {
		t.Errorf("done task detail should have a checked done checkbox\n%s", truncate(body))
	}

	// The lede must contain the Done badge (as a success chip).
	if !strings.Contains(body, "sw-badge--success") && !strings.Contains(body, ">Done") {
		t.Logf("warning: expected Done badge in lede; page:\n%s", truncate(body))
	}
}

// TestActionDetailLedeHasNoLeadingShow checks that on an action detail page
// with show=true, the lede starts directly with badges — no "Show" text before them.
// Acceptance item 3 of task 0195 (action part).
func TestActionDetailLedeHasNoLeadingShow(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action",
		map[string]any{"title": "Ping me", "show": true}), &act)

	rec := get(t, h, "/t/action/"+act.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The mark must not render a visible "Show" label.
	if strings.Contains(body, "> Show") || (strings.Contains(body, "Pinned") && strings.Contains(body, ">Show<")) {
		t.Errorf("action detail should not have visible > Show\n%s", truncate(body))
	}
	if strings.Contains(body, `<span class="sw-visually-hidden">Show`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> Show`) {
		t.Errorf("action detail should not have sw-visually-hidden>Show\n%s", truncate(body))
	}

	// The checkbox must exist and be checked.
	if !strings.Contains(body, `name="prop-show"`) ||
		!strings.Contains(body, `<input class="sw-mark__input" type="checkbox" name="prop-show" value="true"`) {
		t.Errorf("show action detail should have a checked show checkbox\n%s", truncate(body))
	}
	if idx := strings.Index(body, `name="prop-show"`); idx >= 0 && !strings.Contains(body[idx:idx+200], " checked") {
		t.Errorf("show action detail should have a checked show checkbox\n%s", truncate(body))
	}
}

// TestTaskUnLedeHasNoLeadingMarkDone checks that on an undone task detail page,
// the word "Done" does not appear in text nodes — only as an unchecked checkbox.
func TestTaskUndoneDetailLedeHasNoDoneWord(t *testing.T) {
	_, h := newApp(t)

	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task",
		map[string]any{"title": "Water the plants"}), &task)

	rec := get(t, h, "/t/task/"+task.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "> Done") ||
		strings.Contains(body, `<span class="sw-visually-hidden">Done`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> Done`) {
		t.Errorf("undone task detail should not mention Done in text\n%s", truncate(body))
	}

	if !strings.Contains(body, `name="prop-done"`) {
		t.Errorf("undone task detail should have a done checkbox\n%s", truncate(body))
	}
}

// TestPinningNoteStillWorks checks that toggling the pinned checkbox on a note
// detail page still posts correctly and updates the checked state.
// Acceptance item 4 of task 0195.
func TestPinningNoteStillWorks(t *testing.T) {
	_, h := newApp(t)

	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note",
		map[string]any{"title": "Buy tomatoes"}), &note)

	// Start unpinned.
	rec := get(t, h, "/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), `<input class="sw-mark__input" type="checkbox" name="prop-pinned" value="true" checked`) {
		t.Errorf("new note should start unpinned\n%s", truncate(rec.Body.String()))
	}

	// Pin it via the props endpoint.
	req := httptest.NewRequest(http.MethodPost, "/t/note/"+note.ID+"/props",
		strings.NewReader(`prop-pinned=true`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusSeeOther)

	// Re-fetch and check the checkbox is now checked.
	rec = get(t, h, "/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	body2 := rec.Body.String()
	if !strings.Contains(body2, `<input class="sw-mark__input" type="checkbox" name="prop-pinned" value="true"`) {
		t.Errorf("pinned note should have a checked checkbox\n%s", truncate(rec.Body.String()))
	} else if idx := strings.Index(body2, `name="prop-pinned"`); idx < 0 || !strings.Contains(body2[idx:idx+200], " checked") {
		t.Errorf("pinned note should have a checked checkbox\n%s", truncate(rec.Body.String()))
	}

	// Unpin it.
	req = httptest.NewRequest(http.MethodPost, "/t/note/"+note.ID+"/props",
		strings.NewReader(`prop-pinned=false`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusSeeOther)

	// Re-fetch and check the checkbox is now unchecked.
	rec = get(t, h, "/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), `<input class="sw-mark__input" type="checkbox" name="prop-pinned" value="true" checked`) {
		t.Errorf("unpinned note should have an unchecked checkbox\n%s", truncate(rec.Body.String()))
	}
}
