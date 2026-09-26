package server_test

// On a record's own page its box is labelled with its word, where it can be
// seen: Done, Pinned, Show. The word is said once, by the box, and never
// again as a chip or a hidden word beside it; the record is the heading, so
// the box's label does not repeat it.

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
	i := strings.Index(body, `<p class="sw-lede">`)
	if i < 0 {
		t.Fatalf("%s has no lede\n%s", path, truncate(body))
	}
	j := strings.Index(body[i:], "</p>")
	return body[i : i+j]
}

// wantBoxWord checks the lede's box is labelled word, visibly and once, and
// checked or not as it should be.
func wantBoxWord(t *testing.T, lede, field, word string, checked bool) {
	t.Helper()
	input := `<input class="sw-mark__input" type="checkbox" name="prop-` + field + `" value="true"`
	if checked {
		input += " checked>"
	} else {
		input += ">"
	}
	if !strings.Contains(lede, input+" "+word+"</label>") {
		t.Errorf("the %s box should be labelled %q where it can be seen, checked=%v\n%s", field, word, checked, lede)
	}
	if n := strings.Count(lede, word); n != 1 {
		t.Errorf("%q should be said once in the lede, by the box; said %d times\n%s", word, n, lede)
	}
	if strings.Contains(lede, "aria-label") || strings.Contains(lede, `sw-visually-hidden">`+word) || strings.Contains(lede, `sw-visually-hidden"> `+word) {
		t.Errorf("the box is named by its visible label alone\n%s", lede)
	}
}

func TestAPinnedNoteBoxSaysPinned(t *testing.T) {
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy tomatoes", "pinned": true}), &note)
	wantBoxWord(t, ledeOf(t, h, "/t/note/"+note.ID), "pinned", "Pinned", true)
}

func TestAnUnpinnedNoteBoxSaysPinnedUnchecked(t *testing.T) {
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Meeting notes"}), &note)
	wantBoxWord(t, ledeOf(t, h, "/t/note/"+note.ID), "pinned", "Pinned", false)
}

func TestADoneTaskBoxSaysDone(t *testing.T) {
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants", "done": true}), &task)
	wantBoxWord(t, ledeOf(t, h, "/t/task/"+task.ID), "done", "Done", true)
}

func TestAnUndoneTaskBoxSaysDoneUnchecked(t *testing.T) {
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants"}), &task)
	wantBoxWord(t, ledeOf(t, h, "/t/task/"+task.ID), "done", "Done", false)
}

func TestAnActionBoxSaysShow(t *testing.T) {
	_, h := newApp(t)
	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{"title": "Ping me", "show": true}), &act)
	wantBoxWord(t, ledeOf(t, h, "/t/action/"+act.ID), "show", "Show", true)
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
