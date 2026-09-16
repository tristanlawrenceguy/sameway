package look_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// A page reads as its landmarks, headings, controls and live regions, and
// the things a screen reader user would be stuck on are named.
func TestAPageReadsAsAScreenReaderGetsIt(t *testing.T) {
	src := `<!doctype html><html><head><title>Notes</title></head><body>
<header><a href="/">My Sameway</a></header>
<nav aria-label="You are here"><ol><li><a href="/t/note">Notes</a></li></ol></nav>
<main><h1>Water the plants</h1>
<div data-component="record"><h2>Fields</h2><dl><dt>Body</dt><dd>Every Sunday.</dd></dl>
<div class="sw-bar"><form method="post" action="/activity/a1/undo"><input type="hidden" name="from" value="/t/note/x"><button type="submit">Undo</button></form></div></div>
<form action="/chat" method="post"><label for="m">Your message</label><textarea id="m" name="message"></textarea><button type="submit">Send</button></form>
<div id="chat-status" role="status">Ready.</div>
<div hidden><a href="/gone">Not shown</a></div>
</main><footer><a href="/activity">Activity</a></footer></body></html>`
	o, err := look.Page(src)
	if err != nil {
		t.Fatal(err)
	}
	if o.Title != "Notes" || len(o.Problems) != 0 {
		t.Errorf("a clean page has no problems, got title %q problems %v", o.Title, o.Problems)
	}
	roles := []string{}
	for _, l := range o.Landmarks {
		roles = append(roles, l.Role)
	}
	if strings.Join(roles, ",") != "banner,navigation,main,contentinfo" || o.Landmarks[1].Label != "You are here" {
		t.Errorf("landmarks %v", o.Landmarks)
	}
	if len(o.Headings) != 2 || o.Headings[0].Level != 1 || o.Headings[1].Text != "Fields" {
		t.Errorf("headings %v", o.Headings)
	}
	var undo, message, gone *look.Control
	for i := range o.Controls {
		switch o.Controls[i].Name {
		case "Undo":
			undo = &o.Controls[i]
		case "Your message":
			message = &o.Controls[i]
		case "Not shown":
			gone = &o.Controls[i]
		}
	}
	if undo == nil || undo.Kind != "button" || undo.Action != "/activity/a1/undo" || undo.Method != "POST" {
		t.Errorf("the Undo button should say what it submits, got %+v", undo)
	}
	if message == nil || message.Kind != "textbox" || message.Action != "/chat" {
		t.Errorf("the labelled textarea should be a named textbox in its form, got %+v", message)
	}
	if gone == nil || !gone.Hidden {
		t.Errorf("a link under hidden is in the document but not shown, got %+v", gone)
	}
	if len(o.Live) != 1 || o.Live[0].ID != "chat-status" || o.Live[0].Politeness != "polite" || o.Live[0].Text != "Ready." {
		t.Errorf("live regions %v", o.Live)
	}
	if strings.Join(o.Components, ",") != "record" {
		t.Errorf("components %v", o.Components)
	}
}

// The problems are the contract every component is held to, and the rules
// a page must hold on top.
func TestProblemsNameWhatAScreenReaderIsStuckOn(t *testing.T) {
	o, err := look.Page(`<html><head></head><body><h2>Second</h2><h4>Fourth</h4>
<input type="text" name="q"><table><tr><td>1</td></tr></table><img src="x.png">
<span id="dup"></span><span id="dup"></span><p aria-describedby="nope">x</p>
<label for="missing">Name</label><a href="/x"></a><button aria-hidden="true">Hidden</button></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`duplicate id "dup"`, `aria-describedby points at missing id "nope"`, `label for="missing" has no target`,
		"<input> has no label", "table has no caption", "img without alt", "a link with no name",
		"<button> is focusable but aria-hidden", `heading level skips from 2 to 4 at "Fourth"`,
		"0 h1 headings; a page has one", "no main landmark", "no title",
	}
	got := strings.Join(o.Problems, "\n")
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing problem %q in:\n%s", w, got)
		}
	}

	// A fragment is held to the same rules, minus the ones only a page has.
	f, _ := look.Fragment(`<div data-component="card"><h2>Plan</h2><a href="/canvas/b1">Open</a></div>`)
	if len(f.Problems) != 0 {
		t.Errorf("a clean fragment has no problems, got %v", f.Problems)
	}
}
