package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// The sign that points at a change out of sight is for eyes only: aria-hidden,
// never focusable, and gone once the change is seen. Checked in a browser:
// tools/a11y-runner/behave-live.mjs.

func TestOffscreenSignIsBundled(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	js := get(t, h, "/design/sameway.js")
	wantStatus(t, js, http.StatusOK)
	if !strings.Contains(js.Body.String(), "sw-offscreen") {
		t.Error("sameway.js must carry 22-offscreen.js")
	}
	css := get(t, h, "/design/sameway.css")
	wantStatus(t, css, http.StatusOK)
	if !strings.Contains(css.Body.String(), ".sw-offscreen") {
		t.Error("sameway.css must carry 21-offscreen.css")
	}
}

// The page put back where it was at once when it follows a turn, and a
// kept block kept as the same node: behave-refresh.mjs.
