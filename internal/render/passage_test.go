package render

import (
	"strings"
	"testing"
)

// A reply's dash lines reach a screen reader and an agent as a list, not
// as one paragraph with the items run together.
func TestAReplysDashLinesAreAList(t *testing.T) {
	got := string(passage("I can help with:\n- tasks for [tomorrow](/t/task)\n- a shopping list", nil))
	want := `<p>I can help with:</p><ul><li>tasks for <a class="sw-link" href="/t/task">tomorrow</a></li><li>a shopping list</li></ul>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := string(passage("1. first\n2. second", nil)); got != "<ol><li>first</li><li>second</li></ol>" {
		t.Errorf("numbered lines are an ordered list: %s", got)
	}
	if got := string(passage("Costs went up\n- slightly, then down", nil)); !strings.HasPrefix(got, "<p>") || strings.Contains(got, "<li>") != true {
		t.Errorf("a dash line after words is still a list item: %s", got)
	}
	if got := string(passage("one line\n- item\nnot an item", nil)); strings.Contains(got, "<li>") {
		t.Errorf("lines that only start like a list stay a paragraph: %s", got)
	}
}

// Words a model made **bold** are bold, without their stars.
func TestBoldWordsAreBold(t *testing.T) {
	got := string(passage("**Saturday walk** at 9am\n- **Call** grandma", nil))
	want := "<p><strong>Saturday walk</strong> at 9am</p><ul><li><strong>Call</strong> grandma</li></ul>"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
