package server_test

import (
	"strings"
	"testing"
)

// TestCreationTimestampUsesCreated asserts that every record detail page's
// lede shows the creation timestamp without any label text — not "Added" or "Created at".
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

			lower := strings.ToLower(extractWhenMadeSpan(page))
			// The page must NOT contain "created at" label text or nested span.
			if strings.Contains(lower, "created at ") || strings.Contains(lower, "<span") {
				t.Errorf("%s detail page lede should not show \"Created\" label text before the creation timestamp; found \"%s\" instead", rec.typ, extractLabel(page))
			}

			// The page must NOT contain "Added" as a standalone timestamp label.
			if strings.Contains(page, ">Added ") {
				t.Errorf("%s detail page lede should not show bare \"Added\"", rec.typ)
			}

			// The span structure must still exist with a <time> element.
			if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
				t.Errorf("%s detail page lede should have a sw-detail__when span with muted styling", rec.typ)
			}

			if !strings.Contains(extractWhenMadeSpan(page), `<time`) && !strings.Contains(extractWhenMadeSpan(page), `datetime="`) {
				t.Errorf("%s detail page lede should have a <time> element with datetime for the creation timestamp", rec.typ)
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
