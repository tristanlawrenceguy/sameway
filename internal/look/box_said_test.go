package look

import (
	"strings"
	"testing"
)

// A box is read where it sits, [ ] or [x]: "Done" beside an empty box was
// read as a task done.
func TestABoxSaysWhetherItIsTicked(t *testing.T) {
	t.Parallel()
	o, err := Page(`<html><body><main><h1>Essay</h1><div><form><label><input type="checkbox" name="done"> Done</label> <button>Save</button></form> Due today</div><p><label><input type="checkbox" checked> Pinned</label> Added today</p></main></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	for _, p := range o.Text {
		said = append(said, p.Text)
	}
	all := strings.Join(said, " | ")
	if !strings.Contains(all, "[ ] Done") || !strings.Contains(all, "[x] Pinned") {
		t.Errorf("each box says whether it is ticked: %s", all)
	}
}
