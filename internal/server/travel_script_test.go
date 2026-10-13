package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// What a person does moves where it goes (31-travel.js): that items are
// named only for a transition and a refresh compares blocks unnamed is
// checked in a browser (behave-refresh.mjs, and motion.mjs against a
// server, which watches every transition travel or not). That travel is
// short, transform and opacity only, and a cross-fade under reduced motion
// or the still pace: behave-styles.mjs.

// The page's script runs before the page is first drawn, so a page that
// arrives by a view transition names its items in time.
func TestScriptIsReadyBeforeTheFirstDraw(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	res := get(t, h, "/")
	wantStatus(t, res, http.StatusOK)
	if !strings.Contains(res.Body.String(), `<script src="/design/sameway.js" defer blocking="render"></script>`) {
		t.Error("sameway.js must block the first render, so pagereveal is heard")
	}
}
