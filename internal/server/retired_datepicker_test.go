package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// TestRetiredDatepickerBlockShowsAsWhenField checks that a canvas saved
// while datepicker was a component keeps showing its date, now as the
// when-field that replaced it, on the canvas and on the block's own page,
// and that the assistant is no longer offered a datepicker.
func TestRetiredDatepickerBlockShowsAsWhenField(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	blk, err := a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{
		"component": "datepicker",
		"props": map[string]any{
			"label": "Due date", "name": "due", "value": "2026-10-02",
			"min": "2026-09-01", "hint": "From 1 September 2026.", "required": true,
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/canvas/" + blk.ID} {
		rec := get(t, h, path)
		wantStatus(t, rec, http.StatusOK)
		body := rec.Body.String()
		for _, want := range []string{
			`data-component="when-field"`,
			`name="due" type="text" value="2026-10-02"`,
			`type="date" value="2026-10-02"`,
			"From 1 September 2026.",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: want %q in the retired datepicker block", path, want)
			}
		}
		if strings.Contains(body, "could not be shown") {
			t.Errorf("%s: the retired datepicker block showed an error", path)
		}
	}

	if _, ok := a.Registry.Get("datepicker"); ok {
		t.Error("datepicker should not be a component the assistant can choose")
	}
	rec := get(t, h, "/api/describe?full=1")
	wantStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), "datepicker") {
		t.Error("/api/describe still offers datepicker")
	}
}
