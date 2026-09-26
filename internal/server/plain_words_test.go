package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/query"
)

// TestTheDesignPageDrawsEveryComponent: an example the gallery cannot draw
// is a component nobody can see before using it.
func TestTheDesignPageDrawsEveryComponent(t *testing.T) {
	_, h := newApp(t)
	body := get(t, h, "/design").Body.String()
	if i := strings.Index(body, "Could not render"); i >= 0 {
		t.Errorf("the design page failed to draw an example: %.200s", body[i:])
	}
}

// TestConditionsReadAsWords: a list says what it holds the way a person
// would, not in the grammar it was asked in.
func TestConditionsReadAsWords(t *testing.T) {
	a, _ := newApp(t)
	task, _ := a.Types.Get("task")
	for where, want := range map[string]string{
		"done=false":      "not done",
		"done=true":       "done",
		"due<today":       "due before today",
		"due<=+7d":        "due by 7 days from now",
		"due=":            "no due",
		"due!=":           "with a due",
		"title~fern":      "title has fern",
		"tags=garden":     "tags include garden",
		"due>=2026-10-01": "due from Thu 1 Oct 2026",
	} {
		if got := query.Words(task, []string{where}); got != want {
			t.Errorf("%s reads %q, want %q", where, got, want)
		}
	}
}

// TestAnEmptyListSaysWhatItLookedFor: in words, not done=false.
func TestAnEmptyListSaysWhatItLookedFor(t *testing.T) {
	_, h := newApp(t)
	body := get(t, h, "/t/task?where=done%3Dfalse&where=due%3Ctoday").Body.String()
	if !strings.Contains(body, "0 matching not done and due before today") {
		t.Errorf("the list page should say its conditions in words")
	}
}

// TestASearchResultShowsADayNotAStoredTime: the snippet of a task found by
// its title shows its due day as a person reads it.
func TestASearchResultShowsADayNotAStoredTime(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("task", map[string]any{"title": "Repot the fern", "due": "2026-09-27T00:00:00Z"})
	body := get(t, h, "/search?q=fern").Body.String()
	if strings.Contains(body, "2026-09-27T00:00:00Z") || !strings.Contains(body, "Sun 27 Sep 2026") {
		t.Errorf("the result should say Sun 27 Sep 2026, not the stored time")
	}
}
