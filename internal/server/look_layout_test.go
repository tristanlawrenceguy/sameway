package server_test

import (
	"strings"
	"testing"
)

// A look at a tab gives Layout now with its measures, the line the
// assistant is told about it, so an agent reading a tab before changing
// it knows what the assistant would; one block's own page does not.
func TestALookAtATabSaysHowItReads(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	id := storedBlock(t, a, "text", map[string]any{"content": "hello"})
	var looked struct {
		Measured struct {
			Layout string
		}
	}
	decode(t, get(t, h, "/api/look?path=/"), &looked)
	if !strings.HasPrefix(looked.Measured.Layout, "Layout now") || looked.Measured.Layout != a.Chat.LayoutNow("") {
		t.Errorf("a look at Home gives Layout now: %q", looked.Measured.Layout)
	}
	var block struct {
		Measured struct {
			Layout string
		}
	}
	decode(t, get(t, h, "/api/look?path=/canvas/"+id), &block)
	if block.Measured.Layout != "" {
		t.Errorf("one block's page has no layout of a tab: %q", block.Measured.Layout)
	}
}
