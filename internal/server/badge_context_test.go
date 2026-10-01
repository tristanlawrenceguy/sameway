package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A state badge is heard with its field only when the value needs it:
// "Each day" says what it is and is heard alone, never as "Each day cadence".
// Note detail pages do not show status badges in the lede — the definition
// list below carries that information without exposing internal names.
func TestAStateBadgeIsHeardWithItsFieldOnlyWhenItNeedsIt(t *testing.T) {
	a, h := newApp(t)
	habit, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Read", "cadence": "day"})
	note, _ := a.Store.Create("note", map[string]any{"title": "Plans", "status": "draft"})

	page := get(t, h, "/t/habit/"+habit.ID).Body.String()
	if !strings.Contains(page, `>Each day</span>`) || strings.Contains(page, `cadence</span>`) {
		t.Errorf("a named value is heard alone:\n%s", page)
	}
	if page := get(t, h, "/t/note/"+note.ID).Body.String(); strings.Contains(page, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("note detail lede should not expose \"status\" as a hidden field name:\n%s", page)
	}
}

// A file's page calls its facts what a person would, and does not show
// the id its original is stored under.
func TestAFilePageNamesItsFactsInWords(t *testing.T) {
	a, h := newApp(t)
	file, _ := a.Store.Create(server.FileType, map[string]any{"title": "Notes", "name": "notes.txt", "kind": "document", "size": 11, "path": "u7qspudrpgdl2vqu"})

	page := get(t, h, "/t/file/"+file.ID).Body.String()
	for _, want := range []string{"File name", "File type", "File size"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page should say %q", want)
		}
	}
	if strings.Contains(page, "u7qspudrpgdl2vqu") {
		t.Errorf("the stored path should not be shown:\n%s", page)
	}
}

// A file's page shows its size with a unit, not a raw integer. The
// display layer turns 11 into "11 bytes", 2048 into "2 KB", etc., so the
// value reads naturally alongside the label.
func TestAFileShowsItsSizeWithUnitsNotANumber(t *testing.T) {
	a, h := newApp(t)

	// Create a file with exactly 11 bytes — small enough to stay in the "bytes" range.
	file, _ := a.Store.Create(server.FileType, map[string]any{
		"title": "Tiny",
		"name":  "tiny.txt",
		"kind":  "document",
		"size":  11,
		"path":  "u7qspudrpgdl2vqu",
	})

	page := get(t, h, "/t/file/"+file.ID).Body.String()
	if !strings.Contains(page, "11 bytes") {
		t.Errorf("the page should show the size with units:\n%s", page)
	}
}

// A file detail lede does not contain a raw status badge like "Ready".
// The status information is still visible in the definition list below
// but it must not be mashed into the lede paragraph as a standalone chip.
func TestAFileDetailLedeHasNoRawStatusBadge(t *testing.T) {
	a, h := newApp(t)

	// A file with default status "ready".
	file, _ := a.Store.Create(server.FileType, map[string]any{
		"title":  "Ready File",
		"name":   "ready.txt",
		"kind":   "document",
		"status": "ready",
		"path":   "u7qspudrpgdl2vqu",
	})

	page := get(t, h, "/t/file/"+file.ID).Body.String()

	// The lede is the <p class="sw-lede"> element. Check that it does not
	// contain a badge component for "Ready" (which would indicate a raw
	// status value mashed into the meta text).
	if strings.Contains(page, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("the lede should not expose \"status\" as a hidden field name:\n%s", page)
	}

	// Also check that "Ready" does not appear inside the lede paragraph.
	ledeStart := strings.Index(page, `<p class="sw-lede">`)
	if ledeStart == -1 {
		t.Fatal("no lede found in page")
	}
	ledeEnd := strings.Index(page[ledeStart:], `</p>`)
	if ledeEnd == -1 {
		t.Fatal("no closing tag for lede")
	}
	lede := page[ledeStart : ledeStart+ledeEnd]

	// The word "Ready" should not appear as a standalone chip in the lede.
	if strings.Contains(lede, `>Ready</span>`) {
		t.Errorf("the lede must not contain a raw \"Ready\" status badge:\n%s", page)
	}
}
