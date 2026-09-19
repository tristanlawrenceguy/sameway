package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// Checkbox accessible names include checked state so a screen reader user can
// distinguish items without navigating into each control.  Acceptance items 1,
// 2 and 3 of backlog 0333.
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

	// Unchecked: aria-label should be action-oriented.
	if !strings.Contains(page, `aria-label="Mark done Order compost"`) {
		t.Errorf("unchecked task checkbox should have an action-oriented accessible name; page:\n%s", truncate(page))
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

	// Checked: aria-label should be state-describing.
	if !strings.Contains(page, `aria-label="Order compost — done"`) {
		t.Errorf("checked task checkbox should have a state-describing accessible name; page:\n%s", truncate(page))
	}

	// The hidden input toggling the field back to false must still be present.
	if strings.Count(page, `<input type="hidden" name="prop-done" value="false">`) != 1 {
		t.Errorf("the toggle-back hidden input should still exist; page:\n%s", truncate(page))
	}
}

// Pin checkboxes on /t/note lists include checked state in their accessible
// names.  Acceptance items 1 and 3 of backlog 0333.
func TestANotePinCheckboxAccessibleNameReflectsState(t *testing.T) {
	_, h := newApp(t)

	// Create an unpinned note.
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note",
		map[string]any{"title": "Buy tomatoes"}), &note)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection",
		"props":     map[string]any{"type": "note", "label": "Notes"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()

	// Unchecked: aria-label should be action-oriented.
	if !strings.Contains(page, `aria-label="Pin Buy tomatoes"`) {
		t.Errorf("unpinned note checkbox should have an action-oriented accessible name; page:\n%s", truncate(page))
	}

	// Pin the note by posting to its props endpoint.
	req := httptest.NewRequest(http.MethodPost, "/t/note/"+note.ID+"/props",
		strings.NewReader(`prop-pinned=true`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusSeeOther)

	page = get(t, h, "/").Body.String()

	// Checked: aria-label should be state-describing.
	if !strings.Contains(page, `aria-label="Buy tomatoes — pinned"`) {
		t.Errorf("pinned note checkbox should have a state-describing accessible name; page:\n%s", truncate(page))
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
