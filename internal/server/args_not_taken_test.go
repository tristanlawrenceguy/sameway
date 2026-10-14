package server_test

import (
	"strings"
	"testing"
)

// A tool's argument it does not take is refused with what it does take,
// never ignored. agreed is the person's Yes, which only a question's
// answer carries: sent by a caller, it is refused, and deleting a field is
// still asked, never done.
func TestAnArgumentAToolDoesNotTakeIsRefused(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	withPhase(t, a)
	text, isErr := a.Chat.Call("change_field", []byte(`{"type":"note","field":"phase","change":"delete","agreed":true}`))
	if !isErr || !strings.Contains(text, "change_field does not take agreed. It takes change, field, label, type, value") {
		t.Errorf("agreed from a caller is refused, got %v %q", isErr, text)
	}
	if note, _ := a.Types.Get("note"); note != nil {
		if _, ok := note.Field("phase"); !ok {
			t.Error("nothing was deleted without the person's Yes")
		}
	}
	text, isErr = a.Chat.Call("find_records", []byte(`{"type":"note","querry":"plan"}`))
	if !isErr || !strings.Contains(text, "querry (did you mean query?)") {
		t.Errorf("a slip is answered with the one meant, got %v %q", isErr, text)
	}
}
