package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestNonEntryDetailStillShowsCreationTimestamp asserts that non-entry,
// non-reminder record types (task, note) still show the sw-detail__when span
// on their detail pages. The type guard only affects entries and reminders.
// This covers acceptance item 1 for other types: they should continue to work
// as before.
func TestNonEntryDetailStillShowsCreationTimestamp(t *testing.T) {
	t.Parallel()

	records := []struct {
		typ    string
		fields map[string]any
	}{
		{"task", nil},
		{"note", nil},
	}

	for _, rec := range records {
		rec := rec // capture loop var
		t.Run(rec.typ, func(t *testing.T) {
			a, h := newApp(t)

			fields := map[string]any{"title": "Test record"}
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

			// Non-entry types must still show the sw-detail__when span.
			if !strings.Contains(page, `class="sw-detail__when`) {
				t.Errorf("%s detail page should have a sw-detail__when span; got: %s", rec.typ, extractLede(page))
			}

			// The time element inside sw-detail__when must have a datetime attribute.
			re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time datetime="`)
			if !re.MatchString(page) {
				t.Errorf("%s detail page should have <time> with datetime in the whenMade span; got: %s", rec.typ, extractLede(page))
			}
		})
	}
}
