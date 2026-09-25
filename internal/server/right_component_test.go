package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestTheHabitsPageIsTheTracker: habits are where each one stands and a
// press to log, not rows of names with a box that reads as done.
func TestTheHabitsPageIsTheTracker(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("habit", map[string]any{"name": "Water", "target": 8, "unit": "glasses"})
	a.Store.Create("habit", map[string]any{"name": "Old one", "archived": true})
	page := get(t, h, "/t/habit").Body.String()
	for _, want := range []string{`data-component="tracker"`, `role="meter"`, `>Log<`, `>Archived <span class="sw-group__count">1</span>`} {
		if !strings.Contains(page, want) {
			t.Errorf("the habits page should carry %s", want)
		}
	}
	if strings.Contains(page, `type="checkbox"`) {
		t.Errorf("a habit has nothing to tick off")
	}
}

// TestATaskWithNoDayDoesNotSayWhenItChanged: a row says what matters; a
// task's missing due day is not made up for with an edit time.
func TestATaskWithNoDayDoesNotSayWhenItChanged(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("task", map[string]any{"title": "Order compost"})
	a.Store.Create("note", map[string]any{"title": "Garden plan"})
	if strings.Contains(get(t, h, "/t/task").Body.String(), "Updated ") {
		t.Errorf("a task row should not say when it was updated")
	}
	if !strings.Contains(get(t, h, "/t/note").Body.String(), "Updated ") {
		t.Errorf("a note, which has no day of its own, still says when it changed")
	}
}

// TestTheListsOnShowHaveTheirOwnColours: the lists in the navigation each
// get a colour of their own.
func TestTheListsOnShowHaveTheirOwnColours(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("habit", map[string]any{"name": "Water"})
	a.Store.Create("note", map[string]any{"title": "Beds"})
	a.Store.Create("task", map[string]any{"title": "Order compost"})
	page := get(t, h, "/t/note").Body.String()
	nav := page[strings.Index(page, "<nav"):]
	nav = nav[:strings.Index(nav, "</nav>")]
	seen := map[string]string{}
	for _, m := range regexp.MustCompile(`data-dot="(\d)"[^>]*>\s*<a[^>]*href="/t/(\w+)"`).FindAllStringSubmatch(nav, -1) {
		if other, ok := seen[m[1]]; ok {
			t.Errorf("%s and %s share colour %s", other, m[2], m[1])
		}
		seen[m[1]] = m[2]
	}
	if len(seen) < 3 {
		t.Errorf("want three lists with colours in the navigation, found %v in %s", seen, nav)
	}
}
