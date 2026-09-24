package server_test

import (
	"strings"
	"testing"
)

// TestDetailPageHasSwBarInsideBlock checks that every content type record
// detail page renders a <div class="sw-bar sw-quiet"> inside the wrapping
// div with data-block-id. This is the anchor point 08-edit.js looks for on
// line 117 (block.querySelector(".sw-bar")) before inserting its Edit button.
// Acceptance items 1–3: without this bar, clicking "Edit note" does nothing.
func TestDetailPageHasSwBarInsideBlock(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Fatalf("detail page missing data-block-id wrapper\n%s", truncate(body))
	}

	swBarIdx := strings.Index(body[blockOpen:], `<div class="sw-bar sw-quiet">`)
	if swBarIdx < 0 {
		t.Errorf("sw-dl-block for %s should contain a div.sw-bar.sw-quiet inside it\n%s", rec.ID, truncate(body))
	}
}

// TestDetailPageSwBarContainsDeleteLink checks that the Delete link is rendered
// inside the sw-bar (not in a separate cluster), so 08-edit.js finds both the
// Edit anchor and the existing Delete control within the same block. Acceptance
// item 4: the Delete link must still work after being moved into the bar.
func TestDetailPageSwBarContainsDeleteLink(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Fatalf("detail page missing data-block-id wrapper\n%s", truncate(body))
	}

	swBarIdx := strings.Index(body[blockOpen:], `<div class="sw-bar sw-quiet">`)
	if swBarIdx < 0 {
		t.Errorf("cannot check Delete link placement without sw-bar\n%s", truncate(body))
		return
	}

	// Find the closing </div> of the sw-bar (the first one after its opening).
	swBarInner := body[blockOpen+swBarIdx:]
	closeSwBar := strings.Index(swBarInner, `</div>`)
	if closeSwBar < 0 {
		t.Errorf("cannot find end of sw-bar div\n%s", truncate(body))
		return
	}

	barContent := swBarInner[:closeSwBar]
	if !strings.Contains(barContent, "Delete note") {
		t.Errorf("sw-bar should contain the Delete link text \"Delete note\"\nbar content: %s", truncate(barContent))
	}
}

// TestDetailPageActivityHasSwBarInsideBlock checks that activity detail pages
// also render a div.sw-bar.sw-quiet inside their data-block-id wrapper.
func TestDetailPageActivityHasSwBarInsideBlock(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor": "human", "action": "added", "detail": "A test activity",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Fatalf("activity detail missing data-block-id wrapper\n%s", truncate(body))
	}

	swBarIdx := strings.Index(body[blockOpen:], `<div class="sw-bar sw-quiet">`)
	if swBarIdx < 0 {
		t.Errorf("sw-dl-block for activity %s should contain a div.sw-bar.sw-quiet inside it\n%s", rec.ID, truncate(body))
	}
}

// TestDetailPageProposalHasSwBarInsideBlock checks that proposal detail pages
// also render a div.sw-bar.sw-quiet inside their data-block-id wrapper.
func TestDetailPageProposalHasSwBarInsideBlock(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("proposal", map[string]any{
		"summary": "Should we refactor this?", "action": `{"tool":"test"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/proposal/"+rec.ID).Body.String()

	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Fatalf("proposal detail missing data-block-id wrapper\n%s", truncate(body))
	}

	swBarIdx := strings.Index(body[blockOpen:], `<div class="sw-bar sw-quiet">`)
	if swBarIdx < 0 {
		t.Errorf("sw-dl-block for proposal %s should contain a div.sw-bar.sw-quiet inside it\n%s", rec.ID, truncate(body))
	}
}

// TestDetailPageSwBarStructure checks that the definition list comes before the
// sw-bar within the same data-block-id wrapper — dl first, then bar. This is the
// exact structure 08-edit.js expects to find when it scans for [data-prop] elements
// and builds its inline form.
func TestDetailPageSwBarStructure(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note", "tags": []any{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()

	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Fatalf("detail page missing data-block-id wrapper\n%s", truncate(body))
	}

	blockEnd := strings.Index(body[blockOpen:], "</div>")
	if blockEnd < 0 {
		t.Fatalf("cannot find end of sw-dl-block wrapper\n%s", truncate(body))
	}

	blockContent := body[blockOpen : blockOpen+blockEnd]

	dlIdx := strings.Index(blockContent, `<dl class="sw-fields"`)
	barIdx := strings.Index(blockContent, `<div class="sw-bar sw-quiet">`)

	if dlIdx < 0 {
		t.Errorf("sw-dl-block should contain the fields component\n%s", truncate(body))
	}
	if barIdx < 0 {
		t.Errorf("sw-dl-block should contain a div.sw-bar.sw-quiet\n%s", truncate(body))
		return
	}

	if dlIdx > barIdx {
		t.Errorf("dl should come before sw-bar inside the block; found dl at %d and bar at %d relative to block start\n%s", dlIdx, barIdx, truncate(blockContent))
	}
}
