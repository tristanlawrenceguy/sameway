package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// An agent's first read is GET /api/describe. Handed 346 KB with the routes
// last, the REST agent in the evaluation never found how to add a block
// and wrote an HTML page of its own in five tasks out of five. The index
// is small, says how to build first, and documents the block routes whole.
func TestDescribeIsASmallIndexThatSaysHowToBuild(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	t.Logf("index: %d bytes", len(body))
	if len(body) > 16*1024 {
		t.Errorf("the index is %d bytes, more than 16 KB", len(body))
	}
	for _, want := range []string{
		"POST /api/block", "PATCH /api/block/{id}", "DELETE /api/block/{id}", // the build routes
		`\"canvas\"`, `\"span\"`, `\"position\"`, // where a block goes, as the JSON carries it
		"cannot_show", "422", "shows", // the check when written and what it answers
		"Accept: application/json",                   // a page's form answers JSON only when asked
		"never make up records",                      // build from what exists
		"/api/describe/components/{name}", "?full=1", // where the rest is
		"meter: ", "task (", // a line for each component and type
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the index should say %q", want)
		}
	}
	build, routes, components := strings.Index(body, `"build"`), strings.Index(body, `"routes"`), strings.Index(body, `"components"`)
	if !(build < routes && routes < components) {
		t.Errorf("how to build and the routes should come before the components: build %d, routes %d, components %d", build, routes, components)
	}
	if !strings.Contains(body[routes:components], "block_add: POST /api/block") {
		t.Error("the first route should be adding a block")
	}

	// Everything is still there when asked for.
	var full struct {
		Types      []struct{ Name string }
		Components []struct{ Name string }
		Routes     map[string]string
	}
	decode(t, get(t, h, "/api/describe?full=1"), &full)
	if len(full.Types) == 0 || len(full.Components) == 0 || !strings.Contains(full.Routes["block_add"], "shows") {
		t.Error("?full=1 should be the whole description, the block routes among the routes")
	}
	if strings.Contains(full.Routes["mcp_http"], "is off without it") || !strings.Contains(full.Routes["mcp_http"], "sameway agent add") {
		t.Errorf("mcp_http should say an agent key lets a client in: %q", full.Routes["mcp_http"])
	}
}

// One component by name is what writing it takes: the props a writer gives
// and one example, not the props the server fills in, and not the
// accessibility and machine contract, which ?full=1 adds.
func TestDescribeOneComponentIsCompact(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/api/describe/components/collection")
	wantStatus(t, rec, http.StatusOK)
	var c struct {
		Name    string
		Props   struct{ Properties map[string]any }
		Example map[string]any
		Add     string
		A11y    any
	}
	decode(t, rec, &c)
	t.Logf("collection, compact: %d bytes", rec.Body.Len())
	if c.Name != "collection" || c.Props.Properties["type"] == nil || c.Props.Properties["where"] == nil {
		t.Errorf("the collection should come with the props a writer gives, got %+v", c.Props)
	}
	for _, filled := range []string{"items", "groups", "summary", "problem"} {
		if c.Props.Properties[filled] != nil || c.Example[filled] != nil {
			t.Errorf("%s is filled in by the server and should be left out", filled)
		}
	}
	if len(c.Example) == 0 || !strings.Contains(c.Add, "POST /api/block") || c.A11y != nil {
		t.Errorf("one example, how to add it, and no a11y contract: %+v", c)
	}
	whole := get(t, h, "/api/describe/components/collection?full=1").Body.String()
	if !strings.Contains(whole, `"a11y"`) || !strings.Contains(whole, `"items"`) {
		t.Error("?full=1 should give the whole manifest")
	}
	if rec.Body.Len()*2 > len(whole) {
		t.Errorf("compact (%d bytes) should be well under the whole (%d)", rec.Body.Len(), len(whole))
	}
}

// What the index says about adding a block holds: the body it describes
// is written, answers with shows, and lands on the canvas it names.
func TestTheIndexedWayToAddABlockWorks(t *testing.T) {
	_, h := newApp(t)
	rec := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "where": []string{"done=false"}, "label": "To do"},
		"canvas": "", "span": 6, "position": 3,
	})
	wantStatus(t, rec, http.StatusCreated)
	var made struct {
		ID    string
		Shows string
	}
	decode(t, rec, &made)
	if made.ID == "" || made.Shows == "" {
		t.Errorf("a block added as the index says should answer with its id and shows: %+v", made)
	}
	var home struct{ Records []struct{ ID string } }
	decode(t, get(t, h, "/api/block?where=canvas="), &home)
	found := false
	for _, r := range home.Records {
		found = found || r.ID == made.ID
	}
	if !found {
		t.Error("GET /api/block?where=canvas= should list the block on Home, as the index says")
	}
	wantStatus(t, get(t, h, "/api/canvas"), http.StatusOK)
	bad := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "nope"}})
	wantStatus(t, bad, http.StatusUnprocessableEntity)
	if !strings.Contains(bad.Body.String(), `"cannot_show"`) && !strings.Contains(bad.Body.String(), `"invalid"`) {
		t.Errorf("a block that cannot be shown should be refused as the index says: %s", bad.Body.String())
	}
}
