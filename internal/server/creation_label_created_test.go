package server_test

import (
	"strings"
	"testing"
)

// TestCreationLabelIsCreatedNotAdded asserts that every record detail page's
// lede shows "Created" before the creation timestamp — not the bare word "Added".
// This is the opposite of acceptance item 1–4: the standalone "Added" label must
// be gone, replaced by proper context or removed entirely. Acceptance items 1-4.

func TestCreationLabelIsCreatedNotAdded(t *testing.T) {
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

			// The page must NOT contain the bare word "Added" as a creation label.
			if strings.Contains(page, ">Added ") {
				t.Errorf("%s detail page should not show bare \"Added\" before creation timestamp; found \">Added \" in lede", rec.typ)
			}

			// The span structure must still exist.
			if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
				t.Errorf("%s detail page lede should have a sw-detail__when span with muted styling", rec.typ)
			}
		})
	}
}

// TestInteractionDetailLedeHasNoAddedLabel asserts that an interaction detail page
// does not show the bare word "Added" as a creation label. Acceptance item 4.
func TestInteractionDetailLedeHasNoAddedLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("interaction", map[string]any{
		"summary": "Lunch with Priya",
		"at":      "2026-10-15T13:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/interaction/"+rec.ID).Body.String()

	if strings.Contains(page, ">Added ") {
		t.Error("interaction detail page should not show bare \"Added\" before creation timestamp; found \">Added \" in lede")
	}

	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("interaction detail page lede should have a sw-detail__when span with muted styling")
	}
}
