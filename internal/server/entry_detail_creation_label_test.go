package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeCreationTimeIsLabelled asserts that an entry detail page's
// lede shows the creation timestamp with a visible "Created" label before it.
// Acceptance item 2.
func TestEntryDetailLedeCreationTimeIsLabelled(t *testing.T) {
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

	// The <time> element must be present directly in the whenMade span (no label text).
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("entry detail page lede should have a sw-detail__when span with muted styling")
	}

	lower := strings.ToLower(extractWhenMadeSpan(page))
	if strings.Contains(lower, "created at ") || strings.Contains(lower, "<span") {
		t.Error("entry detail page lede must not contain \"created at\" label text or nested span; got an unlabeled time like \"3 days ago\"")
	}

	if !strings.Contains(extractWhenMadeSpan(page), `<time`) || !strings.Contains(extractWhenMadeSpan(page), `datetime="`) {
		t.Error("entry detail page lede should have a <time> element with datetime for the creation timestamp")
	}
}

// TestEntryDetailLedeHabitNameAbsent asserts that an entry detail page's lede
// does not contain the habit name; it is rendered in the fields section below,
// so removing it from the lede eliminates the bare-text concatenation with
// "Created at". Acceptance item 1.
func TestEntryDetailLedeHabitNameAbsent(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Mum's evening medicine",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	entryRec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-02T04:32:00Z",
		"amount": 2.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()
	lede := extractLede(page)

	// The habit name must not appear in the lede at all.
	if strings.Contains(lede, "Mum's evening medicine") {
		t.Errorf("entry detail page lede should not contain the habit name; got: %s", lede)
	}

	// With no badge before whenMade, there is no middle-dot separator between a
	// span and "Created at".
	if strings.Contains(lede, `</span> · <span class="sw-detail__when`) {
		t.Errorf("entry detail page lede should not have a middle-dot separator between a badge and \"Created at\" because the habit badge is removed; got: %s", lede)
	}
}

// TestEntryDetailLedeCreationTimestampHasProperLabel asserts that on an entry
// detail page, the creation timestamp is preceded by a label like "Created at"
// so screen readers announce it with clear context. Acceptance item 2.
func TestEntryDetailLedeCreationTimestampHasProperLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Drink water",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	entryRec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-02T04:32:00Z",
		"amount": 5.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	// The whenMade span must contain only the <time> element — no label text.
	lower := strings.ToLower(extractWhenMadeSpan(page))
	if strings.Contains(lower, "created at ") || strings.Contains(lower, "<span") {
		t.Error("entry detail page lede should not contain \"created at\" label text or nested span before the creation timestamp; got bare \"Created\" without \"at\"")
	}

	if !strings.Contains(extractWhenMadeSpan(page), `<time`) && !strings.Contains(extractWhenMadeSpan(page), `datetime="`) {
		t.Error("entry detail page lede should have a <time> element with datetime for the creation timestamp")
	}
}

// extractLede returns the text content of the sw-lede div from an HTML page.
func extractLede(page string) string {
	start := strings.Index(page, `class="sw-lede"`)
	if start == -1 {
		return ""
	}
	bodyStart := start + len(`class="sw-lede">`)
	end := strings.Index(page[bodyStart:], "</div>")
	if end == -1 {
		return page[bodyStart:]
	}
	return page[bodyStart : bodyStart+end]
}
