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
		{"note", "Meeting notes", nil},
		{"action", "Sync data", nil},
		{"file", "Readme.txt", nil},
	}

	// Task is not tested here — it no longer carries a whenMade span (goals 0121/0341; task 0999).
	// Entry is tested separately (TestEntryDetailNoWhenMadeSpan) because entries
	// do not show a whenMade span — the "Logged …" chip from the glance provides
	// this time information with its label instead.

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

}

// TestEntryDetailNoWhenMadeSpan asserts that entry detail pages do not show a
// separate creation timestamp span — the "Logged …" chip from the glance already
// provides this time information with its label. This is the entry-specific fix:
// only entries skip whenMade to avoid duplicating the same timestamp twice.
func TestEntryDetailNoWhenMadeSpan(t *testing.T) {
	t.Parallel()
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

	// The entry's time is shown only in the "Logged …" glance chip.
	if !strings.Contains(page, ">Logged") {
		t.Error("entry detail page lede should show \"Logged\" from the glance chip")
	}

	// No separate creation timestamp span exists on entry pages.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("entry detail page must not have a sw-detail__when span (creation time is in the Logged chip)")
	}
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
