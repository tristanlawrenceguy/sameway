package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/query"
)

// TestTheDesignPageDrawsEveryComponent: an example the gallery cannot draw
// is a component nobody can see before using it.
func TestTheDesignPageDrawsEveryComponent(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	body := get(t, h, "/design").Body.String()
	if i := strings.Index(body, "could not be shown"); i >= 0 {
		t.Errorf("the design page failed to draw an example: %.200s", body[i:])
	}
}

// TestConditionsReadAsWords: a list says what it holds the way a person
// would, not in the grammar it was asked in.
func TestConditionsReadAsWords(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	_, h := newApp(t)
	body := get(t, h, "/t/task?where=done%3Dfalse&where=due%3Ctoday").Body.String()
	if !strings.Contains(body, "0 matching not done and due before today") {
		t.Errorf("the list page should say its conditions in words")
	}
}

// TestASearchResultShowsADayNotAStoredTime: the snippet of a task found by
// its title shows its due day in natural language, not the stored format.
func TestASearchResultShowsADayNotAStoredTime(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("task", map[string]any{"title": "Repot the fern", "due": "2026-09-27T00:00:00Z"})
	body := get(t, h, "/search?q=fern").Body.String()
	// Natural language dates are used in search snippets now (Relative instead
	// of Text), so no raw RFC 3339 or machine format should appear.
	if strings.Contains(body, "2026-09-27T00:00:00Z") {
		t.Errorf("the result must not contain the stored time; found in:\n%s", truncate(body))
	}
	// The date should be shown in natural language (e.g., weekday, "27 Sep", or
	// a relative form). For a day-only value that is more than 7 days away,
	// shortDay returns "27 Sep" (no year for same-year dates).
	if !strings.Contains(body, "27 Sep") {
		t.Errorf("the result should show the date in natural language; found in:\n%s", truncate(body))
	}
}

// TestABrokenBlockSaysSoInPlainWords: a block whose props do not fit its
// component tells a person what could not be shown and what to do, not
// the schema's own words for why.
func TestABrokenBlockSaysSoInPlainWords(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if _, err := a.Store.Create("block", map[string]any{"component": "button", "props": map[string]any{"text": "Go"}}); err != nil {
		t.Skip("the store will not hold such a block:", err)
	}
	body := get(t, h, "/").Body.String()
	if !strings.Contains(body, "This button could not be shown") {
		t.Errorf("the canvas should say the button could not be shown")
	}
	for _, raw := range []string{"additional properties", "missing property", "Could not render"} {
		if strings.Contains(body, raw) {
			t.Errorf("the canvas should not show %q", raw)
		}
	}
}
