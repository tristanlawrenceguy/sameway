package server_test

import (
	"os"
	"strings"
	"testing"
)

// A reminder is said once, as its name: the list of ringing reminders is
// not an alert (it read the title with every button), a small one beside
// it is, and only the first clock on a page speaks.
func TestAReminderIsSaidOnce(t *testing.T) {
	tpl, _ := os.ReadFile("../../design/components/clock/template.html")
	if strings.Contains(string(tpl), `class="sw-clock__ringing" role="alert"`) || !strings.Contains(string(tpl), `sw-clock__said" role="alert"`) {
		t.Error("the ringing list is not the alert; a small one beside it is")
	}
	js, _ := os.ReadFile("../../design/components/clock/enhance.js")
	if !strings.Contains(string(js), `document.querySelector(".sw-clock .sw-clock__said")`) || !strings.Contains(string(js), `"Reminder: " + d.title`) {
		t.Error("the first clock says the reminder's name, once")
	}
}

// Nothing pulses on its own for more than about five seconds.
func TestNothingPulsesForEver(t *testing.T) {
	for _, p := range []string{"chat", "clock", "status"} {
		css, _ := os.ReadFile("../../design/components/" + p + "/style.css")
		if strings.Contains(string(css), "sw-pulse 1.2s var(--sw-motion-ease) infinite") {
			t.Errorf("%s pulses for ever", p)
		}
	}
}

// Escape puts away controls a pointer revealed.
func TestEscapePutsAwayRevealedControls(t *testing.T) {
	css, _ := os.ReadFile("../../design/base/04-quiet.css")
	js, _ := os.ReadFile("../../design/base/22-quiet.js")
	if !strings.Contains(string(css), `[data-quiet-away]`) || !strings.Contains(string(js), `e.key !== "Escape"`) {
		t.Error("Escape hides what hover revealed")
	}
}

// A note in another language than the workspace's says so on its page.
func TestANoteSaysItsLanguage(t *testing.T) {
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
