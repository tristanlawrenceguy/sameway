package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A list the person wants in reach has its records under its item in the
// sidebar (ui.nest), set by asking and taken back the same way; the record
// open now is the current item, not its list.
func TestAListsRecordsAreNestedInTheSidebarWhenAsked(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	water, err := a.Store.Create(server.HabitType, map[string]any{"name": "Water"})
	if err != nil {
		t.Fatal(err)
	}
	a.Store.Create(server.HabitType, map[string]any{"name": "Old", "archived": true})
	link := `href="/t/` + server.HabitType + `/` + water.ID + `"`

	if page := get(t, h, "/").Body.String(); strings.Contains(page, "sw-nav__sub") {
		t.Error("nothing is nested until asked")
	}
	if err := a.Workspace.Set("ui.nest", "+habit"); err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/").Body.String()
	sub := page[strings.Index(page, `class="sw-nav__sub"`):]
	if !strings.Contains(sub, link) || strings.Contains(sub, ">Old<") {
		t.Errorf("each habit, not an archived one, is under Habits in the sidebar\n%s", page)
	}
	page = get(t, h, "/t/"+server.HabitType+"/"+water.ID).Body.String()
	nav := page[strings.Index(page, `aria-label="Main"`):]
	nav = nav[:strings.Index(nav, "</nav>")]
	if strings.Count(nav, `aria-current="page"`) != 1 {
		t.Errorf("only the habit open now is the current item\n%s", page)
	}
	if err := a.Workspace.Set("ui.nest", "-habit"); err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/").Body.String(); strings.Contains(page, "sw-nav__sub") {
		t.Error("taken back, nothing is nested")
	}
}
