package server_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A press answers at once (32-press.css): in the same frame, with colour,
// a ring and a slight give, never a change of place or size, and under
// reduced motion with no give at all; forced colours still show it.
func TestPressAnswersAtOnceAndMovesNothing(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../design/base/32-press.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	for _, rule := range regexp.MustCompile(`[^{}]*:active[^{}]*\{[^}]*\}`).FindAllString(css, -1) {
		body := rule[strings.Index(rule, "{"):]
		for _, moves := range []string{"width", "height", "margin", "padding", "border-width", "top:", "left:", "font-size"} {
			if strings.Contains(body, moves) {
				t.Errorf("a press must not change place or size (%s): %s", moves, strings.TrimSpace(rule))
			}
		}
	}
	if strings.Count(css, "transition-duration: 0ms") < 2 {
		t.Error("a press answers in the same frame, not after a transition")
	}
	reduced := css[strings.Index(css, "@media (prefers-reduced-motion: reduce)"):]
	if !strings.Contains(reduced[:strings.Index(reduced, "\n}\n")], "transform: none") {
		t.Error("under reduced motion a press does not give")
	}
	if !strings.Contains(css, "@media (forced-colors: active)") || !strings.Contains(css, "Highlight") {
		t.Error("under forced colours a pressed row is outlined in the system colour")
	}
}

// The tick draws itself on the box the mark draws, and under forced colours
// the box, fill and tick take the system's colours, so checked never hangs
// on colours the system has taken away. (The browser's own box would have
// no edge of its own to show in the page's terms.)
func TestMarkTickDrawsAndFallsBack(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../design/components/mark/style.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	for _, want := range []string{
		"appearance: none",
		"border: 2px solid var(--sw-color-border-strong)", // a 3:1 edge
		".sw-mark__input:checked::before { clip-path: inset(0); }",
		"transition: clip-path var(--sw-motion-base)",
		"@media (forced-colors: active)",
		"border-color: CanvasText", // an edge the system draws
		".sw-mark__input:checked { border-color: Highlight; background: Highlight; }",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("mark/style.css must have %s", want)
		}
	}
}

// A tick shown on the press and put back on a refusal is checked in a
// browser against a server: tools/a11y-runner/motion.mjs.
