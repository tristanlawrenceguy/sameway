package server_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// The sign that points at a change out of sight is for eyes only: a
// screen reader and an agent already have each change in the receipt and
// the activity log, so the sign stays out of the accessibility tree and
// the tab order, and is never a button.
func TestOffscreenSignIsSilentToScreenReaders(t *testing.T) {
	data, err := os.ReadFile("../../design/base/19-offscreen.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, `setAttribute("aria-hidden", "true")`) {
		t.Error("the sign must be aria-hidden")
	}
	for _, focusable := range []string{`createElement("button")`, `createElement("a")`, "tabIndex", "tabindex", ".focus("} {
		if strings.Contains(src, focusable) {
			t.Errorf("the sign must not be focusable, found %s", focusable)
		}
	}
	if !strings.Contains(src, "[data-changed]") {
		t.Error("the sign must follow the same change markers as the glow")
	}
}

func TestOffscreenSignIsBundled(t *testing.T) {
	_, h := newApp(t)
	js := get(t, h, "/design/sameway.js")
	wantStatus(t, js, http.StatusOK)
	if !strings.Contains(js.Body.String(), "sw-offscreen") {
		t.Error("sameway.js must carry 19-offscreen.js")
	}
	css := get(t, h, "/design/sameway.css")
	wantStatus(t, css, http.StatusOK)
	if !strings.Contains(css.Body.String(), ".sw-offscreen") {
		t.Error("sameway.css must carry 19-offscreen.css")
	}
}

// When the page follows a turn it puts the person back where they were in
// one step: the page scrolls smoothly, and a smooth correction is seen as
// the page drifting.
func TestRefreshPutsThePageBackInstantly(t *testing.T) {
	data, err := os.ReadFile("../../design/base/17-refresh.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Count(src, `behavior: "instant"`) < 3 {
		t.Error("the page's and the chat log's scroll must be restored instantly")
	}
	if !strings.Contains(src, "moveBefore") {
		t.Error("a kept block should move without leaving the page where the browser allows")
	}
}
