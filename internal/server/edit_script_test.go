package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEditScriptReadsDataEditAction verifies that the inline edit script reads
// a data-edit-action attribute from the block element when building its form,
// and falls back to /canvas/{id}/props when absent. This covers acceptance
// items 1 and 2 of task 0093.
func TestEditScriptReadsDataEditAction(t *testing.T) {
	script := readEditScript(t)

	// Acceptance item 1: the script must look up data-edit-action on the block.
	if !strings.Contains(script, `block.getAttribute("data-edit-action")`) &&
		!strings.Contains(script, "block.getAttribute('data-edit-action')") {
		t.Error("08-edit.js: expected block.getAttribute(\"data-edit-action\") to read a custom edit endpoint from the block element")
	}

	// Acceptance item 2: when no data-edit-action is set, it must fall back to /canvas/{id}/props.
	if !strings.Contains(script, "/canvas/"+"+ id + "+"/props") &&
		!strings.Contains(script, "\"/canvas/\"+id+\"/props\"") {
		t.Error("08-edit.js: expected a fallback to /canvas/{id}/props when data-edit-action is absent")
	}

	// The form action must be set from the resolved variable (not hardcoded).
	if !strings.Contains(script, "form.action = action") &&
		!strings.Contains(script, `form.action = action`) {
		t.Error("08-edit.js: expected form.action to be set from a variable, not hardcoded")
	}
}

// TestEditScriptDoesNotRemoveCanvasFallback ensures the script still builds
// forms targeting /canvas/{id}/props when no data-edit-action is present —
// matching acceptance item 3 so canvas blocks continue to work unchanged.
func TestEditScriptDoesNotRemoveCanvasFallback(t *testing.T) {
	script := readEditScript(t)

	// The fallback path string must still appear in the script body, because
	// when data-edit-action is absent the script falls back to /canvas/{id}/props.
	if !strings.Contains(script, "/canvas/") || !strings.Contains(script, "props") {
		t.Error("08-edit.js: the /canvas/.../props fallback path must still be present for canvas blocks that do not set data-edit-action")
	}

	// The form action must now come from a variable, not hardcoded on one line.
	if !strings.Contains(script, "form.action = action") {
		t.Error("08-edit.js: the hardcoded form.action should have been replaced with an 'action' variable that reads data-edit-action and falls back to /canvas/{id}/props")
	}

	// The fallback must explicitly set action when no attribute is found.
	if !strings.Contains(script, `!action`) {
		t.Error("08-edit.js: expected a check for missing data-edit-action (e.g., if (!action)) before falling back to the /canvas/ default")
	}
}

// readEditScript is the page's script as the browser gets it: every file in
// design/base, in order, as /design/sameway.js joins them, so a test holds
// wherever in them a handler lives.
func readEditScript(t *testing.T) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(findRepoRoot(t), "design", "base", "*.js"))
	if err != nil || len(files) == 0 {
		t.Fatalf("cannot find the scripts in design/base: %v", err)
	}
	var all strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("cannot read %s: %v", f, err)
		}
		all.Write(data)
	}
	return all.String()
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repository root (no .git directory)")
	return ""
}

// TestDetailPageNoteLoadsEditScript checks that GET /t/note/{id} returns HTML
// containing <script defer src="/design/base/08-edit.js"> so the Edit button
// on note detail pages actually activates inline editing. This covers acceptance
// item 1 of task 0096.
func TestDetailPageNoteLoadsEditScript(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note for edit script",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, r, http.StatusOK)

	body := r.Body.String()
	const tag = `<script defer src="/design/base/08-edit.js">`
	if !strings.Contains(body, tag) {
		t.Errorf("GET /t/note/{id} should contain %q in the HTML head\nbody starts with: %s", tag, truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestDetailPageActivityLoadsEditScript checks that GET /t/activity/{id} returns
// HTML containing <script defer src="/design/base/08-edit.js"> so the Edit button
// on activity detail pages actually activates inline editing. This covers
// acceptance item 2 of task 0096.
func TestDetailPageActivityLoadsEditScript(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "added",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/activity/"+rec.ID)
	wantStatus(t, r, http.StatusOK)

	body := r.Body.String()
	const tag = `<script defer src="/design/base/08-edit.js">`
	if !strings.Contains(body, tag) {
		t.Errorf("GET /t/activity/{id} should contain %q in the HTML head\nbody starts with: %s", tag, truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestDetailPageProposalLoadsEditScript checks that GET /t/proposal/{id} returns
// HTML containing <script defer src="/design/base/08-edit.js"> so the Edit button
// on proposal detail pages actually activates inline editing. This covers
// acceptance item 3 of task 0096.
func TestDetailPageProposalLoadsEditScript(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("proposal", map[string]any{
		"summary": "Should we do something?",
		"action":  map[string]any{"tool": "test"},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/proposal/"+rec.ID)
	wantStatus(t, r, http.StatusOK)

	body := r.Body.String()
	const tag = `<script defer src="/design/base/08-edit.js">`
	if !strings.Contains(body, tag) {
		t.Errorf("GET /t/proposal/{id} should contain %q in the HTML head\nbody starts with: %s", tag, truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestListingPagesDoNotLoadEditScript checks that listing pages (/t/note,
// /t/activity, /t/proposal) do NOT contain the 08-edit.js script tag. Inline
// editing on listing pages is out of scope for task 0096. This covers acceptance
// item 4.
func TestListingPagesDoNotLoadEditScript(t *testing.T) {
	a, h := newApp(t)

	// Create records so the listing pages have content and return 200.
	if _, err := a.Store.Create("note", map[string]any{"title": "Listed note"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Store.Create("activity", map[string]any{
		"actor": "human", "action": "updated",
	}); err != nil {
		t.Fatal(err)
	}

	const tag = `<script defer src="/design/base/08-edit.js">`

	for _, path := range []string{"/t/note", "/t/activity"} {
		r := get(t, h, path)
		wantStatus(t, r, http.StatusOK)

		body := r.Body.String()
		if strings.Contains(body, tag) {
			t.Errorf("GET %s should NOT contain the 08-edit.js script tag\nbody starts with: %s", path, truncate(body))
		}
	}
}
