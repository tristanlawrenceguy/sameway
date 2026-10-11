package server_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// mainNav is the sidebar's lists on a page: the Main navigation only.
func mainNav(t *testing.T, page string) string {
	t.Helper()
	i := strings.Index(page, `aria-label="Main"`)
	if i < 0 {
		t.Fatalf("no Main navigation\n%s", page)
	}
	nav := page[i:]
	return nav[:strings.Index(nav, "</nav>")]
}

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
	nav := mainNav(t, get(t, h, "/t/"+server.HabitType+"/"+water.ID).Body.String())
	if strings.Count(nav, `aria-current=`) != 1 || !strings.Contains(nav, link+` aria-current="page"`) {
		t.Errorf("only the habit open now is marked, once\n%s", nav)
	}
	if err := a.Workspace.Set("ui.nest", "-habit"); err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/").Body.String(); strings.Contains(page, "sw-nav__sub") {
		t.Error("taken back, nothing is nested")
	}
}

// On a record's page its list is where the person is, not the page open
// now: aria-current=true, the way GOV.UK marks a section, so a screen
// reader does not say "current page" of the list. On the list's own page
// it is the page. A list whose name starts another's is not marked.
func TestTheSidebarSaysWhereThePersonIsNotThatTheListIsThePage(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	note, err := a.Store.Create("note", map[string]any{"title": "Seeds"})
	if err != nil {
		t.Fatal(err)
	}
	if nav := mainNav(t, get(t, h, "/t/note").Body.String()); !strings.Contains(nav, `href="/t/note" aria-current="page"`) {
		t.Errorf("on the list's page its item is the current page\n%s", nav)
	}
	nav := mainNav(t, get(t, h, "/t/note/"+note.ID).Body.String())
	if !strings.Contains(nav, `href="/t/note" aria-current="true"`) || strings.Contains(nav, `aria-current="page"`) {
		t.Errorf("on a note's page Notes is where the person is, not the page\n%s", nav)
	}
}

// A long nested list stops at twenty with a last link that says how many
// there are and leads to them all, so a cut list never passes for the
// whole; archived records neither count nor take a place.
func TestALongNestedListSaysHowManyAndLeadsToThemAll(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	for i := 0; i < 25; i++ {
		a.Store.Create(server.HabitType, map[string]any{"name": fmt.Sprintf("Old %d", i), "archived": true})
	}
	for i := 0; i < 23; i++ {
		if _, err := a.Store.Create(server.HabitType, map[string]any{"name": fmt.Sprintf("Habit %d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Workspace.Set("ui.nest", "+habit"); err != nil {
		t.Fatal(err)
	}
	nav := mainNav(t, get(t, h, "/").Body.String())
	sub := nav[strings.Index(nav, `class="sw-nav__sub"`):]
	if n := strings.Count(sub, `href="/t/habit/`); n != 20 {
		t.Errorf("twenty habits are named under Habits, got %d\n%s", n, sub)
	}
	if !strings.Contains(sub, `href="/t/habit">See all 23 habits</a>`) || strings.Contains(sub, "Old ") {
		t.Errorf("the last link says how many and leads to the list; archived ones are left out\n%s", sub)
	}
}
