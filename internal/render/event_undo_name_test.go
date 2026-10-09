package render_test

// An Undo on an entry that is itself an undo says what pressing it does,
// "the undo of …", not "undid" a second time.

import (
	"strings"
	"testing"
)

func TestUndoOfAnUndoSaysWhatItUndoes(t *testing.T) {
	t.Parallel()
	reg := builtins(t)
	for action, want := range map[string]string{
		"undid":    "Undo<span class=\"sw-visually-hidden\"> the undo of Assistant changed pace to Calm",
		"put back": "Undo<span class=\"sw-visually-hidden\"> the putting back of Assistant changed pace to Calm",
		"changed":  "Undo<span class=\"sw-visually-hidden\"> changed setting Assistant changed pace to Calm",
	} {
		props := map[string]any{"actor": "human", "action": action, "detail": "Assistant changed pace to Calm", "undo": "/activity/a1/undo"}
		if action == "changed" {
			props["target"] = "setting"
		}
		out, err := reg.Render("event", props)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), want) {
			t.Errorf("%s: want %q in\n%s", action, want, out)
		}
	}
}
