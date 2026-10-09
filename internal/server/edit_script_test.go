package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Where the edit form is sent, a block's own address or its canvas's, is
// checked in a browser: tools/a11y-runner/behave-edit.mjs.

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
// containing <script defer src="/design/base/09-edit.js"> so the Edit button
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
	const tag = `<script defer src="/design/base/09-edit.js">`
	if !strings.Contains(body, tag) {
		t.Errorf("GET /t/note/{id} should contain %q in the HTML head\nbody starts with: %s", tag, truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestDetailPageActivityLoadsEditScript checks that GET /t/activity/{id} returns
// HTML containing <script defer src="/design/base/09-edit.js"> so the Edit button
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
	const tag = `<script defer src="/design/base/09-edit.js">`
	if !strings.Contains(body, tag) {
		t.Errorf("GET /t/activity/{id} should contain %q in the HTML head\nbody starts with: %s", tag, truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestDetailPageProposalLoadsEditScript checks that GET /t/proposal/{id} returns
// HTML containing <script defer src="/design/base/09-edit.js"> so the Edit button
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
	const tag = `<script defer src="/design/base/09-edit.js">`
	if !strings.Contains(body, tag) {
		t.Errorf("GET /t/proposal/{id} should contain %q in the HTML head\nbody starts with: %s", tag, truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestListingPagesDoNotLoadEditScript checks that listing pages (/t/note,
// /t/activity, /t/proposal) do NOT contain the 09-edit.js script tag. Inline
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

	const tag = `<script defer src="/design/base/09-edit.js">`

	for _, path := range []string{"/t/note", "/t/activity"} {
		r := get(t, h, path)
		wantStatus(t, r, http.StatusOK)

		body := r.Body.String()
		if strings.Contains(body, tag) {
			t.Errorf("GET %s should NOT contain the 09-edit.js script tag\nbody starts with: %s", path, truncate(body))
		}
	}
}
