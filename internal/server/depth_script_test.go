package server_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Depth (design/foundations/elevation.md): a card that can be pressed
// lifts without moving anything around it and without rising under
// reduced motion; a skeleton stands in for what is on its way, shimmers
// for under five seconds and not at all under reduced motion or the still
// pace, is hidden from screen readers while its place is busy and says
// what is coming; under forced colours the edges carry what the shadows
// did.
func TestDepthLiftsAndWaitsQuietly(t *testing.T) {
	t.Parallel()
	read := func(name string) string {
		data, err := os.ReadFile("../../design/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	card, skeleton, turn, tokens := read("components/card/style.css"), read("base/35-skeleton.css"), read("base/34-turn.js"), read("tokens/tokens.css")

	for _, want := range []string{"background: var(--sw-color-bg-lift)", "border-color: var(--sw-color-border-lift)", "box-shadow: var(--sw-shadow-2)", "transform: translateY(-1px)"} {
		if !strings.Contains(card, want) {
			t.Errorf("a pressable card lifts with %s", want)
		}
	}
	if !regexp.MustCompile(`prefers-reduced-motion: reduce\)[^\n]*transform: none`).MatchString(card) {
		t.Error("under reduced motion a card does not rise")
	}
	if !regexp.MustCompile(`forced-colors: active\)[^\n]*border-color: Highlight`).MatchString(card) {
		t.Error("under forced colours a lifted card's edge says it")
	}

	// The shimmer: duration times count stays under WCAG 2.2.2's 5 s.
	m := regexp.MustCompile(`animation: sw-shimmer (\d+)ms linear (\d+)`).FindStringSubmatch(skeleton)
	if m == nil {
		t.Fatal("35-skeleton.css: the shimmer must say its duration and a finite count")
	}
	ms, _ := strconv.Atoi(m[1])
	n, _ := strconv.Atoi(m[2])
	if ms*n >= 5000 {
		t.Errorf("the shimmer runs %d ms; it must stop before five seconds", ms*n)
	}
	for _, want := range []string{
		"@media (prefers-reduced-motion: reduce) { .sw-skeleton > span { animation: none !important; } }",
		`:root[data-pace="still"] .sw-skeleton > span { animation: none; }`,
		"background: GrayText",
	} {
		if !strings.Contains(skeleton, want) {
			t.Errorf("35-skeleton.css must have %s", want)
		}
	}
	if regexp.MustCompile(`@keyframes sw-shimmer[^}]*(transform|translate|width|height)`).MatchString(skeleton) {
		t.Error("the shimmer moves light, never the bars")
	}
	for _, want := range []string{
		`li.setAttribute("aria-busy", "true")`,
		`<div class="sw-skeleton" aria-hidden="true">`,
		`blk.setAttribute("aria-busy", "true")`,
		`b.removeAttribute("aria-busy")`,
	} {
		if !strings.Contains(turn, want) {
			t.Errorf("34-turn.js must have %s", want)
		}
	}
	for _, want := range []string{"--sw-shadow-3:", "--sw-color-bg-lift:", "--sw-color-border-lift:"} {
		if strings.Count(tokens, want) < 2 {
			t.Errorf("tokens.css must define %s in light and dark", want)
		}
	}
}
