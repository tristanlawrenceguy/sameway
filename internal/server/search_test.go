package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// A person searches on one page and lands on what they found; an agent
// gets the same hits as JSON; the page reads cleanly to a screen reader.
func TestOneSearchOverEverything(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Garden", "body": "Weed the beds."}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()
	if !strings.Contains(body, "Call the plumber") || strings.Contains(body, "Garden") || !strings.Contains(body, "1 thing found") {
		t.Errorf("the page should show the one hit and say so: %.400s", body)
	}
	if !strings.Contains(body, `role="search"`) || !strings.Contains(body, `value="plumber"`) {
		t.Error("the search form is a search landmark and keeps the words typed")
	}
	if o, _ := look.Page(body); len(o.Problems) != 0 {
		t.Errorf("the search page should read cleanly, got %v", o.Problems)
	}
	if empty := get(t, h, "/search?q=zebra").Body.String(); !strings.Contains(empty, "Nothing has zebra in it") {
		t.Error("no hits should say so in plain words")
	}
	if footer := get(t, h, "/").Body.String(); !strings.Contains(footer, `href="/search"`) {
		t.Error("search should be reachable from every page")
	}

	var out struct {
		Count int
		Hits  []struct{ Type, Title, Href, Snippet string }
	}
	decode(t, get(t, h, "/api/search?q=kitchen"), &out)
	if out.Count != 1 || out.Hits[0].Title != "Call the plumber" || !strings.Contains(out.Hits[0].Snippet, "kitchen") {
		t.Errorf("the API should give the same hit with its snippet, got %+v", out)
	}
}
