package server_test

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// What a person does moves where it goes (31-travel.js): that items are
// named only for a transition and a refresh compares blocks unnamed is
// checked in a browser (behave-refresh.mjs, and motion.mjs against a
// server, which watches every transition travel or not).

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
