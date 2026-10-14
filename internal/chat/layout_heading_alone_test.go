package chat_test

import (
	"strings"
	"testing"
)

// A section heading with nothing under it, as when what it named was
// taken away, is said in Layout now; one with its section under it is not.
func TestLayoutNowSaysAHeadingWithNothingUnderIt(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	run(t, svc, "add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Garden"}, "span": 12, "frame": "bare"})
	run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Beds to dig"}, "span": 12})
	run(t, svc, "add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Reading"}, "span": 12, "frame": "bare"})
	text := svc.LayoutNow("")
	if !strings.Contains(text, `the section heading "Reading" has nothing under it`) {
		t.Errorf("the last heading has nothing under it:\n%s", text)
	}
	if strings.Contains(text, `"Garden" has nothing under it`) {
		t.Errorf("Garden has its list under it:\n%s", text)
	}
}
