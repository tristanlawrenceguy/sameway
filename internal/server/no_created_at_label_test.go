package server_test

import (
	"strings"
	"testing"
)

// TestNoCreatedAtLabelInWhenMadeSpan asserts that the creation time line on
// record detail pages does not contain any bare "created at", "Created at", or
// "Created" label text — visible or visually-hidden. The <time> element with
// the relative timestamp must still be present.
// Acceptance item 1–2: no visible label, timestamp preserved.
func TestNoCreatedAtLabelInWhenMadeSpan(t *testing.T) {
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
		{"file", "Readme.txt", nil},
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
			if strings.Contains(lower, ">created") || strings.Contains(lower, "<span") {
				t.Errorf("%s detail page must not contain a bare \"Created\" or sw-visually-hidden span in the whenMade area; got:\n%s", rec.typ, whenMadeContent)
			}

			// The <time> element must still be present with human-readable text.
			if !strings.Contains(whenMadeContent, "<time") || !strings.Contains(whenMadeContent, `datetime="`) {
				t.Errorf("%s detail page lede should have a <time> element in the whenMade span; got:\n%s", rec.typ, whenMadeContent)
			}

			// The outer span structure must still exist.
			if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
				t.Errorf("%s detail page lede should have a sw-detail__when span with muted styling", rec.typ)
			}
		})
	}

	// Entry type: habit is needed so we can create an entry.
	t.Run("entry", func(t *testing.T) {
		a, h := newApp(t)

		habitRec, err := a.Store.Create("habit", map[string]any{
			"name":   "Exercise",
			"target": 1.0,
		})
		if err != nil {
			t.Fatal(err)
		}

		entryRec, err := a.Store.Create("entry", map[string]any{
			"habit":  habitRec.ID,
			"at":     "2026-10-02T04:32:00Z",
			"amount": 3.0,
		})
		if err != nil {
			t.Fatal(err)
		}

		page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()
		whenMadeContent := extractWhenMadeSpan(page)

		lower := strings.ToLower(whenMadeContent)
		if strings.Contains(lower, "created at ") {
			t.Errorf("entry detail page must not contain \"created at\" in the whenMade area; got:\n%s", whenMadeContent)
		}
		if strings.Contains(lower, ">created") || strings.Contains(lower, "<span") {
			t.Errorf("entry detail page must not contain a bare \"Created\" or sw-visually-hidden span in the whenMade area; got:\n%s", whenMadeContent)
		}

		if !strings.Contains(whenMadeContent, "<time") || !strings.Contains(whenMadeContent, `datetime="`) {
			t.Errorf("entry detail page lede should have a <time> element in the whenMade span; got:\n%s", whenMadeContent)
		}

		if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
			t.Error("entry detail page lede should have a sw-detail__when span with muted styling")
		}
	})
}

// TestInteractionDetailNoCreatedAtLabel asserts that an interaction detail page's
// lede does not contain a "created at" label between its "Happened..." time chip
// and its creation timestamp. This addresses the screen-reader orphan problem:
// without a visually-hidden wrapper, two <time> elements with no context between
// them confuse assistive technology. The fix removes the label entirely so only
// the single creation-time <time> remains.
func TestInteractionDetailNoCreatedAtLabel(t *testing.T) {
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
	whenMadeContent := extractWhenMadeSpan(page)

	lower := strings.ToLower(whenMadeContent)
	if strings.Contains(lower, "created at ") {
		t.Errorf("interaction detail page must not contain \"created at\" in the whenMade area; got:\n%s", whenMadeContent)
	}
	if strings.Contains(lower, ">created") || strings.Contains(lower, "<span") {
		t.Errorf("interaction detail page must not contain a sw-visually-hidden span or bare label in the whenMade area; got:\n%s", whenMadeContent)
	}

	if !strings.Contains(whenMadeContent, "<time") || !strings.Contains(whenMadeContent, `datetime="`) {
		t.Errorf("interaction detail page lede should have a <time> element in the whenMade span; got:\n%s", whenMadeContent)
	}

	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("interaction detail page lede should have a sw-detail__when span with muted styling")
	}
}

// TestPersonDetailNoCreatedAtLabel asserts that a person detail page's lede does not
// contain a "created at" label between glance facts and the creation timestamp.
func TestPersonDetailNoCreatedAtLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("person", map[string]any{
		"name": "Priya",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/person/"+rec.ID).Body.String()
	whenMadeContent := extractWhenMadeSpan(page)

	lower := strings.ToLower(whenMadeContent)
	if strings.Contains(lower, "created at ") {
		t.Errorf("person detail page must not contain \"created at\" in the whenMade area; got:\n%s", whenMadeContent)
	}
	if strings.Contains(lower, ">created") || strings.Contains(lower, "<span") {
		t.Errorf("person detail page must not contain a sw-visually-hidden span or bare label in the whenMade area; got:\n%s", whenMadeContent)
	}

	if !strings.Contains(whenMadeContent, "<time") || !strings.Contains(whenMadeContent, `datetime="`) {
		t.Errorf("person detail page lede should have a <time> element in the whenMade span; got:\n%s", whenMadeContent)
	}

	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("person detail page lede should have a sw-detail__when span with muted styling")
	}
}

// TestEventDetailNoCreatedAtLabel asserts that an event (task) detail page's lede
// does not contain a "created at" label. The <time> element for the creation time
// must still be present and correctly formatted.
func TestEventDetailNoCreatedAtLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("event", map[string]any{
		"title": "Team standup",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/event/"+rec.ID).Body.String()
	whenMadeContent := extractWhenMadeSpan(page)

	lower := strings.ToLower(whenMadeContent)
	if strings.Contains(lower, "created at ") {
		t.Errorf("event detail page must not contain \"created at\" in the whenMade area; got:\n%s", whenMadeContent)
	}
	if strings.Contains(lower, ">created") || strings.Contains(lower, "<span") {
		t.Errorf("event detail page must not contain a sw-visually-hidden span or bare label in the whenMade area; got:\n%s", whenMadeContent)
	}

	if !strings.Contains(whenMadeContent, "<time") || !strings.Contains(whenMadeContent, `datetime="`) {
		t.Errorf("event detail page lede should have a <time> element in the whenMade span; got:\n%s", whenMadeContent)
	}

	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("event detail page lede should have a sw-detail__when span with muted styling")
	}
}
