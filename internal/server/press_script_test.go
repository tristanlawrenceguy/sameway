package server_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A press answers at once (27-press.css): in the same frame, with colour,
// a ring and a slight give, never a change of place or size, and under
// reduced motion with no give at all; forced colours still show it.
func TestPressAnswersAtOnceAndMovesNothing(t *testing.T) {
	data, err := os.ReadFile("../../design/base/27-press.css")
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

// A tick is shown on the press, before the server answers, and a refusal
// puts the box and the row back; the words said are still the server's.
func TestMarkShowsThePressAndRollsBack(t *testing.T) {
	data, err := os.ReadFile("../../design/base/13-mark.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	change := src[strings.Index(src, `box.addEventListener("change"`):]
	if strings.Index(change, "shown();") > strings.Index(change, "send();") {
		t.Error("the row is struck through on the press, before the save is sent")
	}
	if !strings.Contains(src, "if (failed) box.checked = !sent;\n          shown();") {
		t.Error("a refused save puts the box and the row back")
	}
	if !strings.Contains(src, "say(fresh, failed)") {
		t.Error("what is said is the server's outcome")
	}
}
