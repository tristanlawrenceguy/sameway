package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// A checkbox is named with its field and its record, so a screen reader
// user can tell one row's box from the next, and the name stays put when it
// is ticked: the checked state is the box's own to say.
func TestAMarkCheckboxAccessibleNameReflectsState(t *testing.T) {
	_, h := newApp(t)

	// Create an unchecked task so the done checkbox is unchecked.
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task",
		map[string]any{"title": "Order compost"}), &task)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection",
		"props":     map[string]any{"type": "task", "label": "Tasks"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()

	// The name is the field and the record, whatever the state: the box
	// itself says checked or not checked, so the name never says it twice or
	// changes under the person as they tick it.
	name := `> Done<span class="sw-visually-hidden"> Order compost</span>`
	if !strings.Contains(page, name) || strings.Contains(page, `class="sw-mark__input" type="checkbox" name="prop-done" value="true" aria-label`) {
		t.Errorf("unchecked task checkbox should be named %q by its label, with no aria-label; page:\n%s", name, truncate(page))
	}
	// Checked state attribute must still be absent on the unchecked input.
	if strings.Count(page, `<input class="sw-mark__input" type="checkbox" name="prop-done" value="true"`) != 1 {
		t.Errorf("the single task checkbox should not have checked=; page:\n%s", truncate(page))
	}

	// Mark the task done and re-fetch.
	req := httptest.NewRequest(http.MethodPost, "/t/task/"+task.ID+"/props",
		strings.NewReader(`prop-done=true`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusSeeOther)

	page = get(t, h, "/").Body.String()

	// Checked: the same name, and the box checked.
	if !strings.Contains(page, name) || !strings.Contains(page, `value="true" checked>`) {
		t.Errorf("checked task checkbox should keep its name %q and be checked; page:\n%s", name, truncate(page))
	}

	// The hidden input toggling the field back to false must still be present.
	if strings.Count(page, `<input type="hidden" name="prop-done" value="false">`) != 1 {
		t.Errorf("the toggle-back hidden input should still exist; page:\n%s", truncate(page))
	}
}

// A note's pin is a setting, not something done: its row has no box that
// reads as done, and a pinned note says Pinned in words.
func TestAPinnedNoteSaysSoWithoutACheckbox(t *testing.T) {
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note",
		map[string]any{"title": "Buy tomatoes", "pinned": true}), &note)
	page := get(t, h, "/t/note").Body.String()
	if strings.Contains(page, `type="checkbox"`) {
		t.Errorf("a note's row should have no checkbox; page: %s", truncate(page))
	}
	if !strings.Contains(page, ">Pinned<") {
		t.Errorf("a pinned note should say Pinned; page: %s", truncate(page))
	}
}

// The mark component renders an aria-label on its input when given one,
// overriding the implicit label-based accessible name for screen readers.
func TestAMarkComponentRendersAriaLabel(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Render("mark", map[string]any{
		"type":      "task",
		"record":    "abc123",
		"field":     "done",
		"label":     "Done",
		"context":   "Order compost",
		"checked":   false,
		"ariaLabel": "Mark done Order compost",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, `aria-label="Mark done Order compost"`) {
		t.Errorf("mark with ariaLabel should render it on the input; got:\n%s", got)
	}
	// The visible label text must still be present.
	if !strings.Contains(got, "> Done") {
		t.Error("visible label text must not change when aria-label is added")
	}

	out2, err := reg.Render("mark", map[string]any{
		"type":      "note",
		"record":    "def456",
		"field":     "pinned",
		"label":     "Pinned",
		"context":   "Buy tomatoes",
		"checked":   true,
		"ariaLabel": "Buy tomatoes — pinned",
	})
	if err != nil {
		t.Fatal(err)
	}
	got2 := string(out2)

	if !strings.Contains(got2, `aria-label="Buy tomatoes — pinned"`) {
		t.Errorf("checked mark with ariaLabel should render it; got:\n%s", got2)
	}
}
