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
	for _, want := range []string{"File name", "File type", "Size in bytes"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page should say %q", want)
		}
	}
	if strings.Contains(page, "u7qspudrpgdl2vqu") {
		t.Errorf("the stored path should not be shown:\n%s", page)
	}
}
