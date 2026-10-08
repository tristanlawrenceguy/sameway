package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// A link to another site opens in a new tab and says so, wherever it is
// written: the key page offered with no model, and a link in a note; a
// link inside Sameway stays in its tab.
func TestLinksToOtherSitesOpenInANewTab(t *testing.T) {
	a, h := newApp(t)
	note, _ := a.Store.Create("note", map[string]any{"title": "Recipe", "body": "From [the blog](https://example.com/pancakes), see also [my list](/t/note)."})
	for _, path := range []string{"/", "/t/note/" + note.ID} {
		page := get(t, h, path).Body.String()
		for _, a := range regexp.MustCompile(`<a [^>]*href="https?://[^"]*"[^>]*>`).FindAllString(page, -1) {
			if !strings.Contains(a, `target="_blank"`) || !strings.Contains(a, `rel="noopener"`) && !strings.Contains(a, `rel="opener"`) {
				t.Errorf("%s: a link elsewhere stays in this tab: %s", path, a)
			}
		}
		if strings.Contains(page, `href="/t/note" target=`) {
			t.Errorf("%s: a link inside Sameway opens a new tab", path)
		}
	}
	if page := get(t, h, "/t/note/"+note.ID).Body.String(); !strings.Contains(page, "the blog<span class=\"sw-visually-hidden\"> (opens in a new tab)</span>") {
		t.Errorf("a screen reader hears it opens a new tab: %s", truncate(page))
	}
}
