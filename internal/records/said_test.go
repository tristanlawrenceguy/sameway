package records

import (
	"strings"
	"testing"
)

// What someone said has its JSON taken out before the line is cut: closed,
// nested, an array, or cut off before it closed; braces that are not JSON
// stay, and so do the words around.
func TestSaidWordsTakeOutJSONFirst(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		`add a task using {"type": "task", "title": "x"} please`:                                  "add a task using [content omitted] please",
		`use props {"type": "task", "props": {"where": ["done=false"]}, "done": false} then stop`: "use props [content omitted] then stop",
		`ids ["a1", "b2"] are gone`:                    "ids [content omitted] are gone",
		`keep {maybe} and [later] as they are`:         "keep {maybe} and [later] as they are",
		`a quote "}" inside {"t": "a } b", "u": 1} ok`: `a quote "}" inside [content omitted] ok`,
	} {
		if got := SaidWords(in, 200); got != want {
			t.Errorf("SaidWords(%q)\n got %q\nwant %q", in, got, want)
		}
	}
	// Long enough to be cut inside its JSON: the JSON goes first, so no
	// half of it is left.
	long := "use the add_component tool with props " + `{"type": "task", "title": "test", "done": false, "where": ["due<today"]}` + " and then call GET /api/tasks?where=done=true"
	if got := SaidWords(long, 80); strings.Contains(got, `"`) || !strings.Contains(got, omitted) {
		t.Errorf("cut after the JSON is taken out: %q", got)
	}
	// What was stored cut before, by an older version, is cleaned when said.
	if got := stripJSONLike(`use props {"type": "task", "title": "test", "done":…`); got != "use props "+omitted {
		t.Errorf("JSON cut before it closed goes to the end: %q", got)
	}
}
