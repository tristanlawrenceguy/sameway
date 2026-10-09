package server_test

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// What a person does moves where it goes (31-travel.js): items are named
// only for the moment of a transition, so the page at rest is the server's
// and a refresh still compares blocks as sent; nothing here moves focus.
func TestTravelNamesOnlyForTheTransition(t *testing.T) {
	data, err := os.ReadFile("../../design/base/31-travel.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{
		`removeProperty("view-transition-name")`, // names come off again
		`addEventListener("pageswap"`,            // the page left names its items
		`addEventListener("pagereveal"`,          // and the page arrived names them too
		"e.viewTransition.finished",              // until the transition is over
		"e.persisted",                            // a page back from the cache is as sent
		"MOST",                                   // a long list is not all lifted at once
	} {
		if !strings.Contains(src, want) {
			t.Errorf("31-travel.js must have %s", want)
		}
	}
	if strings.Contains(src, ".focus(") {
		t.Error("travel must never move focus")
	}
	refresh, _ := os.ReadFile("../../design/base/19-refresh.js")
	if !strings.Contains(string(refresh), "travel.clear(); merge(doc); if (travel) travel.name();") {
		t.Error("a refresh unnames items before it compares blocks, and names the new ones after")
	}
}

// Travel is transform and opacity only, short, in tokens, and under
// reduced motion or the still pace it is a cross-fade where things land:
// nothing moves, grows or slides.
func TestTravelIsShortAndStillWhenAskedTo(t *testing.T) {
	data, err := os.ReadFile("../../design/base/30-travel.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	if regexp.MustCompile(`\d+m?s\b`).MatchString(regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(css, "")) {
		t.Error("durations in 30-travel.css come from the motion tokens")
	}
	for _, kf := range regexp.MustCompile(`@keyframes [^{]+\{[^}]*\}[^}]*\}|@keyframes [^{]+\{[^}]*\}`).FindAllString(css, -1) {
		for _, prop := range regexp.MustCompile(`([a-z-]+):`).FindAllStringSubmatch(kf, -1) {
			if prop[1] != "opacity" && prop[1] != "transform" {
				t.Errorf("keyframes animate transform and opacity only: %s", kf)
			}
		}
	}
	reduced := css[strings.Index(css, "@media (prefers-reduced-motion: reduce)"):]
	if strings.Contains(reduced[:strings.Index(reduced, "\n}\n")], "sw-days") {
		t.Error("under reduced motion no transition slides")
	}
	for _, want := range []string{
		`::view-transition-group(*) { animation: none !important; }`,
		`:root[data-pace="still"]::view-transition-group(*)`,
		`:root[data-pace="quick"] { --sw-travel: var(--sw-motion-fast); }`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("30-travel.css must have %s", want)
		}
	}
	tokens, _ := os.ReadFile("../../design/tokens/tokens.json")
	if !strings.Contains(string(tokens), `"base": "220ms"`) || !strings.Contains(string(tokens), `"fast": "120ms"`) {
		t.Error("travel takes motion-base (220ms) and motion-fast (120ms), under a quarter of a second")
	}
}

// The page's script runs before the page is first drawn, so a page that
// arrives by a view transition names its items in time.
func TestScriptIsReadyBeforeTheFirstDraw(t *testing.T) {
	_, h := newApp(t)
	res := get(t, h, "/")
	wantStatus(t, res, http.StatusOK)
	if !strings.Contains(res.Body.String(), `<script src="/design/sameway.js" defer blocking="render"></script>`) {
		t.Error("sameway.js must block the first render, so pagereveal is heard")
	}
}
