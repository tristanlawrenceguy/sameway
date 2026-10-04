package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestActionDetailLedeNoStartedLabel asserts that the action detail page lede
// shows the creation time without any label prefix such as "Started".
// Acceptance item 3.
func TestActionDetailLedeNoStartedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{"title": "Sync data"})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/action/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("action detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Started" (a label prefix).
	if strings.HasPrefix(whenText, "Started ") {
		t.Errorf("action detail lede should not have \"Started\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("action detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestEntryDetailLedeNoAddedLabel asserts that the entry detail page lede shows
// the creation time without any label prefix such as "Added". Acceptance item 3.
func TestEntryDetailLedeNoAddedLabel(t *testing.T) {
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

	body := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("entry detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Added" (a label prefix).
	if strings.HasPrefix(whenText, "Added ") {
		t.Errorf("entry detail lede should not have \"Added\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("entry detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestFileDetailLedeNoAddedLabel asserts that the file detail page lede shows
// the creation time without any label prefix such as "Added". Acceptance item 3.
func TestFileDetailLedeNoAddedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("file", map[string]any{"title": "Readme"})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/file/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("file detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Added" (a label prefix).
	if strings.HasPrefix(whenText, "Added ") {
		t.Errorf("file detail lede should not have \"Added\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("file detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestPersonDetailLedeNoAddedLabel asserts that the person detail page lede shows
// the creation time without any label prefix such as "Added". Acceptance item 3.
func TestPersonDetailLedeNoAddedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("person", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/person/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("person detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Added" (a label prefix).
	if strings.HasPrefix(whenText, "Added ") {
		t.Errorf("person detail lede should not have \"Added\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("person detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestProjectDetailLedeNoAddedLabel asserts that the project detail page lede shows
// the creation time without any label prefix such as "Added". Acceptance item 3.
func TestProjectDetailLedeNoAddedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{"title": "My Project"})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/project/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("project detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Added" (a label prefix).
	if strings.HasPrefix(whenText, "Added ") {
		t.Errorf("project detail lede should not have \"Added\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("project detail lede should show a relative timestamp; got %q", whenText)
	}
}
