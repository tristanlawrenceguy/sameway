package server_test

// On a record's own page its box is labelled with its word via aria-label:
// Done, Pinned, Show. The word is said once, by the checkbox in its lede; no
// bare visible text node repeats it (task 0317).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ledeOf is the line under a record page's title.
func ledeOf(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	rec := get(t, h, path)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	i := strings.Index(body, `<div class="sw-lede">`)
	if i < 0 {
		t.Fatalf("%s has no lede\n%s", path, truncate(body))
	}
	j := strings.Index(body[i:], "</div>")
	return body[i : i+j]
}

// wantBoxWord checks the lede's box is labelled word via aria-label and once,
// and checked or not as it should be. Detail pages use quiet+ariaLabel so the
// label text is invisible; the checkbox name comes from aria-label (task 0317).
func wantBoxWord(t *testing.T, lede, field, word string, checked bool) {
	t.Helper()

	// The input must exist with correct attributes. Order: value, aria-label, checked.
	input := `<input class="sw-mark__input" type="checkbox" name="prop-` + field + `" value="true"`
	if !strings.Contains(lede, input) {
		t.Errorf("the %s box should have the checkbox input element\n%s", field, lede)
	}
	if !strings.Contains(lede, ` aria-label="`+word+`"`) {
		t.Errorf("the %s box should have aria-label=%q\n%s", field, word, lede)
	}
	if checked && !strings.Contains(lede, " checked") {
		t.Errorf("the %s box should be checked when checked=%v\n%s", field, checked, lede)
	}

	// The word must appear exactly once: in the aria-label attribute.
	if n := strings.Count(lede, word); n != 1 {
		t.Errorf("%q should be said once in the lede (aria-label), not %d times\n%s", word, n, lede)
	}

	// No visible label text node after the input.
	if strings.Contains(lede, "> "+word+"</label>") {
		t.Errorf("the %s box should not have visible label text in lede\n%s", field, lede)
	}
}

func TestAPinnedNoteBoxSaysPinned(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy tomatoes", "pinned": true}), &note)
	wantBoxWord(t, ledeOf(t, h, "/t/note/"+note.ID), "pinned", "Pinned", true)
}

func TestAnUnpinnedNoteBoxSaysPinnedUnchecked(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Meeting notes"}), &note)
	wantBoxWord(t, ledeOf(t, h, "/t/note/"+note.ID), "pinned", "Pinned", false)
}

func TestADoneTaskBoxSaysDone(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants", "done": true}), &task)
	wantBoxWord(t, ledeOf(t, h, "/t/task/"+task.ID), "done", "Done", true)
}

func TestAnUndoneTaskBoxSaysDoneUnchecked(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants"}), &task)
	wantBoxWord(t, ledeOf(t, h, "/t/task/"+task.ID), "done", "Done", false)
}

func TestAnActionBoxSaysItsLabel(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{"title": "Ping me", "show": true}), &act)
	wantBoxWord(t, ledeOf(t, h, "/t/action/"+act.ID), "show", "On the canvas", true)
}

// TestPinningNoteStillWorks checks that toggling the pinned checkbox on a note
// detail page still posts correctly and updates the checked state.
// Acceptance item 4 of task 0195.
func TestPinningNoteStillWorks(t *testing.T) {
	t.Parallel()
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
