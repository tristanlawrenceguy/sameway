package server_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A turn you can watch, without noise (34-turn.js and .css). The status
// line says each step in words, politely, no more often than it can be
// heard and each thing once; the reply's words grow a few times a second
// outside any live region and are heard once, whole; a block about to be
// added or changed is outlined in its place in the assistant's colour and
// its arrival plays on from there; nothing travels, and under reduced
// motion and the still pace nothing waits.
func TestATurnIsSaidOnceAndMarkedInPlace(t *testing.T) {
	read := func(name string) string {
		data, err := os.ReadFile("../../design/base/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	js, css, live := read("34-turn.js"), read("33-turn.css"), read("17-live.js")
	for _, want := range []string{
		"var GAP = 2500",                          // one thing said at a time, with time to hear it
		"if (said[words]) return",                 // each thing once a turn
		`sw.status(status, "working", words, "")`, // in the region already on the page, drawn as heard
		`"data-still"`,                            // a long step says it is still going, in its own words
		`"Writing the reply…"`,                    // the reply starting is said once, not its words
		"setTimeout(flush, 80)",                   // words land a few times a second, not each piece
		`behavior: "instant"`,                     // the log does not glide after every word
		`"sw-block sw-block--pending"`,            // a new block's place is held
		`"data-actor", "assistant"`,               // in the assistant's colour
		`"data-pending"`,                          // a block being changed is outlined
		"sw-visually-hidden",                      // the held place says what is coming
	} {
		if !strings.Contains(js, want) {
			t.Errorf("34-turn.js must have %s", want)
		}
	}
	if strings.Contains(js, "aria-live") || strings.Contains(js, "role=\"alert\"") {
		t.Error("34-turn.js must say things through the status already there, not a region of its own")
	}
	for _, want := range []string{
		`sw.watchTurn(form)`,
		"watch.step(d, label)",
		"watch.done()",
		`canvas.querySelector(".sw-block--pending")`, // the block lands where its place was held
		`"sw-landed"`,
	} {
		if !strings.Contains(live, want) {
			t.Errorf("17-live.js must have %s", want)
		}
	}
	if regexp.MustCompile(`live\.words\.data \+=`).MatchString(live) {
		t.Error("17-live.js must not put each piece of the reply on the page as it comes")
	}
	for _, moves := range []string{"transform", "translate", "scale(", "width:", "height:"} {
		for _, rule := range regexp.MustCompile(`@keyframes[^{]*\{[^}]*\}`).FindAllString(css, -1) {
			if strings.Contains(rule, moves) {
				t.Errorf("a held place only fades in, never %s: %s", moves, rule)
			}
		}
	}
	for _, want := range []string{
		"outline: 2px dashed var(--sw-actor, var(--sw-color-assistant))", // the arrival's first stage, held
		"[data-pending]",
		"@media (prefers-reduced-motion: reduce)",
		`.sw-landed > :not(.sw-visually-hidden) { animation: sw-hold var(--sw-motion-fast) linear both !important; }`, // reduced: a quick cross-fade
		`:root[data-pace="still"] .sw-block--pending { animation: none; }`,                                            // still: just there
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("33-turn.css must have %s", want)
		}
	}
}
