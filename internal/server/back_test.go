package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// An action comes back where it was taken: every form in a canvas block
// says its block with no script, a closer place sent after it wins, only
// an id is taken, and the page returns there.
func TestAnActionComesBackWhereItWasTaken(t *testing.T) {
	a, h := newApp(t)
	task, err := a.Store.Create("task", map[string]any{"title": "Order compost"})
	if err != nil {
		t.Fatal(err)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"},
	}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	i := strings.Index(page, `id="block-`)
	if i < 0 {
		t.Fatal("each block on the canvas has an id to come back to")
	}
	block := page[i+len(`id="`) : i+strings.Index(page[i+4:], `"`)+4]
	if !strings.Contains(page, `<input type="hidden" name="back" value="`+block+`">`) {
		t.Errorf("the block's forms say they are in %s", block)
	}

	path := "/t/task/" + task.ID + "/props"
	send := func(back ...string) string {
		res := doWithReferer(t, h, path, url.Values{"prop-done": {"true"}, "back": back}, "http://example.com/")
		return res.Header().Get("Location")
	}
	if loc := send(block); loc != "/#"+block {
		t.Errorf("a tick in a block comes back to it, not %q", loc)
	}
	if loc := send(block, "row-7"); loc != "/#row-7" {
		t.Errorf("the closest place, sent last, wins, not %q", loc)
	}
	if loc := send(`x" onload="y`); loc != "/" {
		t.Errorf("only an id is taken, not %q", loc)
	}
}

// A saved edit says what changed, from what to what, by what a page shows
// for it: a ref by its title, a long text only that it changed.
func TestASavedEditSaysWhatChanged(t *testing.T) {
	a, h := newApp(t)
	garden, _ := a.Store.Create("project", map[string]any{"title": "Garden"})
	house, _ := a.Store.Create("project", map[string]any{"title": "House"})
	task, err := a.Store.Create("task", map[string]any{"title": "Order compost", "project": garden.ID})
	if err != nil {
		t.Fatal(err)
	}
	path := "/t/task/" + task.ID + "/props"
	for _, c := range []struct {
		field, value, want string
	}{
		{"project", house.ID, "Project changed from Garden to House."},
		{"project", "", "Project cleared; it was House."},
		{"title", "Order more compost", "Title changed from Order compost to Order more compost."},
		{"notes", "Two bags.", "Notes changed."},
	} {
		res := doWithReferer(t, h, path, url.Values{"prop-" + c.field: {c.value}}, "http://example.com"+"/t/task/"+task.ID)
		if body := after(t, h, res).Body.String(); !strings.Contains(body, c.want) || !strings.Contains(body, "Changes saved") {
			t.Errorf("editing %s should say %q under Changes saved", c.field, c.want)
		}
	}
}
