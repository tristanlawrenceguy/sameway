package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// Every action a person takes ends the same way: back on the page they
// were on, told once what happened, with Undo in the message when it can
// be taken back.
func TestEveryActionSaysWhatHappenedWhereThePersonIs(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Water the plants"})

	// Saved from a tab: back on that tab, told.
	r := withReferer(t, h, http.MethodPost, "/t/note/"+rec.ID+"/props", "/c/garden", url.Values{"prop-title": {"Water the garden"}}.Encode(), "application/x-www-form-urlencoded")
	page, at := landed(t, h, r)
	if at != "/c/garden" || !strings.Contains(page.Body.String(), "Changes saved") {
		t.Errorf("a save returns to the tab it was made on and says so, got %q", at)
	}

	// Once: the same page again says nothing.
	again := httptest.NewRequest(http.MethodGet, at, nil)
	for _, c := range page.Result().Cookies() {
		again.AddCookie(c)
	}
	reload := httptest.NewRecorder()
	h.ServeHTTP(reload, again)
	if strings.Contains(reload.Body.String(), "sw-outcome") {
		t.Error("a reload does not say it again")
	}

	// Undo, from the message itself.
	undo := regexp.MustCompile(`<form method="post" action="(/activity/[^"]+/undo)" class="sw-outcome__undo"><input type="hidden" name="from" value="([^"]+)">`).FindStringSubmatch(page.Body.String())
	if undo == nil {
		t.Fatalf("the message carries Undo; body: %s", truncate(page.Body.String()))
	}
	back := after(t, h, postForm(t, h, undo[1], url.Values{"from": {undo[2]}}))
	if !strings.Contains(back.Body.String(), "Undone") {
		t.Error("undoing says so")
	}
	if got, _ := a.Store.Get("note", rec.ID); got.Fields["title"] != "Water the plants" {
		t.Errorf("undo from the message puts it back, got %v", got.Fields["title"])
	}

	// Only ever back to a page here.
	r = postForm(t, h, "/activity/nothing/undo", url.Values{"from": {"//elsewhere.example/"}})
	if loc := r.Header().Get("Location"); loc != "/" {
		t.Errorf("a way back that leaves the site is not taken, got %q", loc)
	}
	if !strings.Contains(after(t, h, r).Body.String(), "Not undone") {
		t.Error("an undo that cannot be done says so")
	}
}

// A block whose component the workspace no longer has is refused in
// words, not with a crash.
func TestABlockWithAGoneComponentIsRefusedInWords(t *testing.T) {
	a, h := newApp(t)
	blk, err := a.Store.Create("block", map[string]any{"component": "nothing-like-it", "props": map[string]any{}})
	if err != nil {
		t.Skip("the store will not hold such a block:", err)
	}
	r := postForm(t, h, "/canvas/"+blk.ID+"/props", url.Values{"prop-title": {"x"}})
	if body := after(t, h, r).Body.String(); !strings.Contains(body, "no longer has") {
		t.Errorf("the refusal says why; body: %s", truncate(body))
	}
}
