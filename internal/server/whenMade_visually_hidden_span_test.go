package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestWhenMadeWrapsCreatedAtInVisuallyHiddenSpan asserts that whenMade() wraps
// "Created at" inside a <span class="sw-visually-hidden"> so screen readers
// hear it as context for the following timestamp rather than an orphaned label.
// Acceptance item 2.
func TestWhenMadeWrapsCreatedAtInVisuallyHiddenSpan(t *testing.T) {
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

			// The whenMade span must contain the visually-hidden "created at" label.
			if !strings.Contains(page, `class="sw-visually-hidden">`) {
				t.Errorf("%s detail page should wrap \"Created at\" in a sw-visually-hidden span; got:\n%s", rec.typ, extractWhenMadeSpan(page))
			}

			// The label must be lowercase inside the visually-hidden span.
			if !strings.Contains(strings.ToLower(extractWhenMadeSpan(page)), "created at ") {
				t.Errorf("%s detail page should contain \"created at \" (lowercase) inside a sw-visually-hidden span; got:\n%s", rec.typ, extractWhenMadeSpan(page))
			}

			// The <time> element must follow the visually-hidden span.
			re := regexp.MustCompile(`<span class="sw-detail__when[^"]*"><span class="sw-visually-hidden">created at </span><time datetime="`)
			if !re.MatchString(page) {
				t.Errorf("%s detail page should have <time> immediately after the sw-visually-hidden \"created at\" span; got:\n%s", rec.typ, extractWhenMadeSpan(page))
			}

			// The outer whenMade structure must still exist.
			if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
				t.Errorf("%s detail page should have a sw-detail__when span with muted styling; got:\n%s", rec.typ, extractWhenMadeSpan(page))
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
