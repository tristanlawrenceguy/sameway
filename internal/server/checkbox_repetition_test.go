package server_test

// On record detail pages checkbox fields must not repeat their value as a
// bare text node alongside metadata labels like "Created at". The mark
// component uses quiet+ariaLabel so the label is in an aria-attribute on
// the input instead of visible after it (task 0317).

import (
	"net/http"
	"strings"
	"testing"
)

func ledeHTML(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	rec := get(t, h, path)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	i := strings.Index(body, `<div class="sw-lede">`)
	if i < 0 {
		t.Fatalf("%s has no lede", path)
	}
	j := strings.Index(body[i:], "</div>")
	return body[i : i+j]
}

// wantNoVisibleBoxLabel asserts the checkbox value does NOT appear as a bare
// visible text node after the input element in the lede. The label must be
// provided via aria-label instead (task 0317, acceptance items 1-3).
func wantNoVisibleBoxLabel(t *testing.T, lede, field, word string) {
	t.Helper()

	input := `<input class="sw-mark__input" type="checkbox" name="prop-` + field + `" value="true"`

	// The bug: a bare space followed by the label text after the input.
	if strings.Contains(lede, input+" "+word+"</label>") {
		t.Errorf("the %s checkbox should not repeat its value as visible text in lede (acceptance item 1-3)\n%s", field, lede)
	}

	// The fix: aria-label must carry the name.
	if !strings.Contains(lede, "aria-label=\""+word+"\"") {
		t.Errorf("the %s checkbox should have aria-label=%q for accessibility (acceptance item 4)\n%s", field, word, lede)
	}

	// The value must not appear as a separate visible text node at all.
	// Count occurrences of the bare word outside attribute values.
	if strings.Contains(lede, "> "+word+"<") || strings.Contains(lede, "> "+word+"</span>") {
		t.Errorf("the %s checkbox should have no bare text node with its value (acceptance item 1-3)\n%s", field, lede)
	}

	// A Save button inside the mark renders its own aria-label when quiet;
	// check that it does not reintroduce visible label text.
	if strings.Contains(lede, ">Save "+word) {
		t.Errorf("the save button should not repeat the checkbox value as visible text\n%s", lede)
	}
}

func TestADoneTaskDetailHasNoVisibleBoxLabel(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants", "done": true}), &task)
	lede := ledeHTML(t, h, "/t/task/"+task.ID)
	wantNoVisibleBoxLabel(t, lede, "done", "Done")
}

func TestAnUndoneTaskDetailHasNoVisibleBoxLabel(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants"}), &task)
	lede := ledeHTML(t, h, "/t/task/"+task.ID)
	wantNoVisibleBoxLabel(t, lede, "done", "Done")
}

func TestAPinnedNoteDetailHasNoVisibleBoxLabel(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy tomatoes", "pinned": true}), &note)
	lede := ledeHTML(t, h, "/t/note/"+note.ID)
	wantNoVisibleBoxLabel(t, lede, "pinned", "Pinned")
}

func TestAnUnpinnedNoteDetailHasNoVisibleBoxLabel(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Meeting notes"}), &note)
	lede := ledeHTML(t, h, "/t/note/"+note.ID)
	wantNoVisibleBoxLabel(t, lede, "pinned", "Pinned")
}

func TestAnActionDetailHasNoVisibleBoxLabel(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{"title": "Ping me", "show": true}), &act)
	lede := ledeHTML(t, h, "/t/action/"+act.ID)
	wantNoVisibleBoxLabel(t, lede, "show", "On the canvas")
}

// wantMetadataPresent asserts that standard metadata fields still appear in
// the lede after the checkbox fix (acceptance item 5).
func wantMetadataPresent(t *testing.T, lede string) {
	t.Helper()
	if !strings.Contains(lede, `class="sw-detail__when"`) && !strings.Contains(lede, "Created at") {
		t.Errorf("the lede should still show creation time metadata (acceptance item 5)\n%s", lede)
	}
}

func TestADoneTaskDetailMetadataStillAppears(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water the plants", "done": true}), &task)
	lede := ledeHTML(t, h, "/t/task/"+task.ID)
	wantMetadataPresent(t, lede)
}

func TestAPinnedNoteDetailMetadataStillAppears(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy tomatoes"}), &note)
	lede := ledeHTML(t, h, "/t/note/"+note.ID)
	wantMetadataPresent(t, lede)
}
