package server_test

import (
	"strings"
	"testing"
)

// A reminder said once, as its name, by the first clock; nothing pulsing for
// ever; and Escape putting away what the pointer revealed: checked in a
// browser, tools/a11y-runner/behave-components.mjs and behave-styles.mjs.

// A note in another language than the workspace's says so on its page.
func TestANoteSaysItsLanguage(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	note, err := a.Store.Create("note", map[string]any{"title": "Einkauf", "body": "Milch und Brot.", "language": "de"})
	if err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/t/note/"+note.ID).Body.String(); !strings.Contains(page, `data-edit-action="/t/note/`+note.ID+`/props" lang="de">`) {
		t.Errorf("the note's words are marked German; body: %s", truncate(page))
	}
	odd, _ := a.Store.Create("note", map[string]any{"title": "X", "language": `de" onmouseover="x`})
	if page := get(t, h, "/t/note/"+odd.ID).Body.String(); strings.Contains(page, `onmouseover="x`) || strings.Contains(page, ` lang="de`) {
		t.Error("only a language code goes into the page")
	}
}
