package server_test

import (
	"strings"
	"testing"
)

// TestWhenMadeHasNoBareCreatedAtTextNode asserts that no bare text node "Created at"
// appears between time elements on record detail pages — it is always wrapped in a
// <span class="sw-visually-hidden">. Acceptance item 1.
func TestWhenMadeHasNoBareCreatedAtTextNode(t *testing.T) {
	t.Parallel()

	records := []struct {
		typ    string
		title  string
		fields map[string]any
	}{
		{"task", "Buy milk", nil},
		{"note", "Meeting notes", nil},
		{"reminder", "Take medicine", map[string]any{"at": "2026-10-09T06:00:00Z"}},
		{"action", "Sync data", nil},
		{"file", "Readme", nil},
	}

	for _, rec := range records {
		t.Run(rec.typ, func(t *testing.T) {
			a, h := newApp(t)

			fields := map[string]any{"title": rec.title}
			if rec.fields != nil {
				for k, v := range rec.fields {
					fields[k] = v
				}
			}
			r, err := a.Store.Create(rec.typ, fields)
			if err != nil {
				t.Fatal(err)
			}

			page := get(t, h, "/t/"+rec.typ+"/"+r.ID).Body.String()

			// The page must NOT contain bare "Created at" text between time elements.
			// After the fix, "created at" is inside a sw-visually-hidden span, not
			// as a bare text node before <time>.
			if strings.Contains(page, ">Created at ") {
				t.Errorf("%s detail page must not contain bare \"Created at\" text node; it should be wrapped in a sw-visually-hidden span:\n%s", rec.typ, extractWhenMadeSpan(page))
			}

			// The visually-hidden span must still convey the label to screen readers.
			if !strings.Contains(page, `class="sw-detail__when`) {
				t.Errorf("%s detail page should have a sw-detail__when span:\n%s", rec.typ, extractWhenMadeSpan(page))
			}

			// The whenMade content must still contain the time element.
			if !strings.Contains(page, `<time datetime="`) {
				t.Errorf("%s detail page must contain a <time> element for creation timestamp;\n%s", rec.typ, extractWhenMadeSpan(page))
			}
		})
	}
}
