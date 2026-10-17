package server_test

import (
	"strings"
	"testing"
)

// TestCreationLabelIsCreatedNotAdded asserts that every record detail page's
// lede shows "Created" before the creation timestamp — not the bare word "Added".
func TestCreationTimestampUsesCreated(t *testing.T) {
	t.Parallel()

	records := []struct {
		typ    string
		title  string
		fields map[string]any
	}{
		{"task", "Buy milk", nil},
		{"note", "Meeting notes", nil},
		{"reminder", "Take medicine", map[string]any{"at": "2026-10-09T06:00:00Z"}},
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

			// The page must contain "Created" before the creation timestamp.
			if !strings.Contains(page, ">Created ") {
				t.Errorf("%s detail page lede should show \"Created\" before the creation timestamp; found \"%s\" instead of a natural label", rec.typ, extractLabel(page))
			}

			// The page must NOT contain "Added" as a standalone timestamp label.
			if strings.Contains(page, ">Added ") {
				t.Errorf("%s detail page lede should not show bare \"Added\" — it now uses \"Created\"", rec.typ)
			}

			// The span structure must still exist.
			if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
				t.Errorf("%s detail page lede should have a sw-detail__when span with muted styling", rec.typ)
			}
		})
	}
}

// extractLabel pulls the visible label text from the sw-detail__when span.
func extractLabel(page string) string {
	start := strings.Index(page, `class="sw-detail__when`)
	if start == -1 {
		return "(no span found)"
	}
	end := strings.Index(page[start:], `</span>`)
	if end == -1 {
		return "(unclosed span)"
	}
	return page[start : start+end]
}
