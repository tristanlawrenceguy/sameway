package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestWhenMadeHasNoBareCreatedAtLabel asserts that whenMade() does not include
// any "created at" label text — visible or visually-hidden. The <time> element
// with the relative timestamp is preserved directly inside the outer span.
// Acceptance item 1–2: no bare label, timestamp preserved.
func TestWhenMadeHasNoBareCreatedAtLabel(t *testing.T) {
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
			whenMadeContent := extractWhenMadeSpan(page)

			// The whenMade span must not contain any form of "created at" label.
			lower := strings.ToLower(whenMadeContent)
			if strings.Contains(lower, "created at ") {
				t.Errorf("%s detail page must not contain \"created at\" in the whenMade area; got:\n%s", rec.typ, whenMadeContent)
			}
			if strings.Contains(lower, "<span") {
				t.Errorf("%s detail page must not contain a nested <span> in the whenMade area; got:\n%s", rec.typ, whenMadeContent)
			}

			// The <time> element must be directly inside the outer span.
			re := regexp.MustCompile(`<span class="sw-detail__when[^"]*"><time datetime="`)
			if !re.MatchString(page) {
				t.Errorf("%s detail page should have <time> directly inside the sw-detail__when span; got:\n%s", rec.typ, whenMadeContent)
			}

			// The outer whenMade structure must still exist.
			if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
				t.Errorf("%s detail page should have a sw-detail__when span with muted styling; got:\n%s", rec.typ, whenMadeContent)
			}
		})
	}
}

// extractWhenMadeSpan returns the content of the sw-detail__when span from an HTML page.
func extractWhenMadeSpan(page string) string {
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
