package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A block set up wrong says so, in one way everywhere, rather than show
// an empty month or "Nothing tracked yet", which read as nothing there.
func TestABlockSetUpWrongSaysWhatIsWrong(t *testing.T) {
	a, h := newApp(t)
	if _, err := a.Store.Create("habit", map[string]any{"name": "Walk", "tags": []any{"health"}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		component string
		props     map[string]any
		said, why string
	}{
		{"calendar", map[string]any{"type": "event"}, "This calendar cannot be shown", "there is no content type event"},
		{"calendar", map[string]any{"type": "note"}, "This calendar cannot be shown", "note has no date field to place on a calendar"},
		{"calendar", map[string]any{"type": "task", "date": "title"}, "This calendar cannot be shown", `task has no date field &#34;title&#34;; its date fields are due`},
		{"calendar", map[string]any{"type": "task", "where": []string{"owner=me"}}, "This calendar cannot be shown", "owner"},
		{"tracker", map[string]any{"label": "Garden", "tags": []string{"garden"}}, "This list of habits cannot be shown as it is set up", "no habit is tagged garden; the habits have health"},
		{"collection", map[string]any{"type": "event"}, "This list cannot be shown", "there is no content type event"},
	} {
		res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": c.component, "props": c.props})
		wantStatus(t, res, http.StatusCreated)
		var blk struct{ ID string }
		decode(t, res, &blk)
		page := get(t, h, "/canvas/"+blk.ID).Body.String()
		if !strings.Contains(page, `data-component="problem"`) || !strings.Contains(page, c.said) || !strings.Contains(page, c.why) {
			t.Errorf("%s %v should say %q and, under What is wrong, %q", c.component, c.props, c.said, c.why)
		}
	}
}
