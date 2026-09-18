package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A person sees the lists that have something in them, and any list they
// made themselves; an empty list the system provides stays out of the
// way, and so do the links meant for builders, unless the workspace asks.
func TestASidebarShowsWhatHasSomethingInIt(t *testing.T) {
	a, h := newApp(t)
	page := get(t, h, "/").Body.String()
	for _, empty := range []string{`href="/t/file"`, `href="/t/action"`, `href="/t/task"`} {
		if strings.Contains(page, empty) {
			t.Errorf("an empty provided list is not in the sidebar, found %s", empty)
		}
	}
	if strings.Contains(page, `href="/design"`) || strings.Contains(page, `href="/api/describe">For agents`) {
		t.Error("the builder links stay out of a person's way by default")
	}
	if !strings.Contains(page, `<link rel="describedby" href="/api/describe">`) {
		t.Error("an agent still finds the guide from the page's head")
	}

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost"}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `href="/t/task"`) || strings.Contains(page, `href="/t/file"`) {
		t.Error("a list appears the moment it has something in it")
	}

	// A list the person made is theirs to see before anything is in it.
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/types", map[string]any{"name": "habit", "fields": []map[string]any{{"name": "name", "type": "string", "required": true}}}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `href="/t/habit"`) {
		t.Error("a list the person made shows even when empty")
	}

	// The workspace can ask for everything.
	a.Workspace.Config.UI.Lists, a.Workspace.Config.UI.Developer = "all", "shown"
	page = get(t, h, "/").Body.String()
	if !strings.Contains(page, `href="/t/file"`) || !strings.Contains(page, `href="/design"`) || !strings.Contains(page, `href="/api/describe">For agents`) {
		t.Error("ui.lists: all and ui.developer: shown bring every list and the builder links back")
	}
}
