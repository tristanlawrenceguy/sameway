package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// seeds is a workspace where "seeds" is in two notes and three tasks.
func seeds(t *testing.T) http.Handler {
	t.Helper()
	_, h := newApp(t)
	for _, title := range []string{"Order seeds", "Sow the seeds"} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": title}), http.StatusCreated)
	}
	for _, title := range []string{"Buy seeds", "Water seeds", "Label seeds"} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": title}), http.StatusCreated)
	}
	return h
}

// Search covers everything; the kinds found are offered as links with how
// many each found, All first and marked as the one shown.
func TestSearchIsEverythingWithTheKindsCounted(t *testing.T) {
	h := seeds(t)
	body := get(t, h, "/search?q=seeds").Body.String()
	for _, want := range []string{"Order", "Sow the", "Buy", "Water", "Label", "5 things found",
		`<nav class="sw-tabs" data-component="tabs" aria-label="Kinds of result">`,
		`href="/search?q=seeds" aria-current="page">All (5)</a>`,
		`href="/search?q=seeds&amp;type=task">Tasks (3)</a>`,
		`href="/search?q=seeds&amp;type=note">Notes (2)</a>`,
		"<title>Search: seeds, 5 results"} {
		if !strings.Contains(body, want) {
			t.Errorf("everything is searched and counted by kind; missing %q\n%s", want, truncate(body))
		}
	}
	if strings.Index(body, "Tasks (3)") > strings.Index(body, "Notes (2)") {
		t.Error("the kind that found most comes first after All")
	}
	for _, kind := range []string{"Habits (", "Messages (", "Blocks (", "Activities ("} {
		if strings.Contains(body, kind) {
			t.Errorf("a kind with nothing found, or one kept by the system, is not offered: %s", kind)
		}
	}
	if o, _ := look.Page(body); len(o.Problems) != 0 {
		t.Errorf("the results read cleanly, got %v", o.Problems)
	}
	// One kind found is the same list as All: no row saying so.
	if one := get(t, h, "/search?q=water").Body.String(); strings.Contains(one, "Kinds of result") {
		t.Error("no row of kinds when only one kind was found")
	}
}

// ?type narrows the results, said once in the heading and the window title,
// with the way back to everything first.
func TestSearchNarrowedToAKind(t *testing.T) {
	h := seeds(t)
	body := get(t, h, "/search?q=seeds&type=task").Body.String()
	if strings.Contains(body, "Order") || !strings.Contains(body, "Buy") || !strings.Contains(body, "Label") {
		t.Errorf("only tasks are shown\n%s", truncate(body))
	}
	if !strings.Contains(body, "<h1>Search: seeds in tasks</h1>") || !strings.Contains(body, "<title>Search: seeds in tasks, 3 results") {
		t.Errorf("the heading and title say the kind\n%s", truncate(body))
	}
	if n := strings.Count(body, "in tasks"); n != 2 {
		t.Errorf("the kind is said once in the heading and once in the title, got %d", n)
	}
	scope := strings.Index(body, `Showing tasks only, 3 of 5 found. <a class="sw-link" href="/search?q=seeds">Search everything</a>`)
	if scope < 0 || scope > strings.Index(body, "<h2>Results</h2>") || scope > strings.Index(body, `<form method="get" action="/search"`) {
		t.Errorf("the scope and the way to everything come first\n%s", truncate(body))
	}
	if !strings.Contains(body, `aria-current="page">Tasks (3)</a>`) || !strings.Contains(body, `href="/search?q=seeds">All (5)</a>`) {
		t.Errorf("the kind shown is the current one, All widens\n%s", truncate(body))
	}
	if !strings.Contains(body, `<input type="hidden" name="type" value="task">`) || !strings.Contains(body, ">Search tasks") {
		t.Error("searching again stays in tasks, and the box says so")
	}
	if o, _ := look.Page(body); len(o.Problems) != 0 {
		t.Errorf("the narrowed results read cleanly, got %v", o.Problems)
	}
	// A kind with none of the words offers everything, and says how many.
	none := get(t, h, "/search?q=sow&type=task").Body.String()
	if !strings.Contains(none, "Nothing in tasks matches “sow”, but 1 thing elsewhere does.") || !strings.Contains(none, `href="/search?q=sow">search everything</a>`) {
		t.Errorf("an empty kind points to everything\n%s", truncate(none))
	}
	if !strings.Contains(none, "<title>Search: sow in tasks, no results") {
		t.Error("the title says the kind found nothing")
	}
}

// From a kind's own pages, Search opens narrowed to that kind with the way
// out first; from anywhere general it is everything.
func TestSearchFromAKindsPlaceStartsThere(t *testing.T) {
	h := seeds(t)
	var out struct{ Hits []struct{ Href string } }
	decode(t, get(t, h, "/api/search?q=buy"), &out)
	for _, from := range []string{"/t/task", out.Hits[0].Href} {
		if page := get(t, h, from).Body.String(); !strings.Contains(page, `href="/search?type=task"`) {
			t.Errorf("Search from %s is a search of tasks", from)
		}
	}
	for _, from := range []string{"/", "/chat", "/activity"} {
		if page := get(t, h, from).Body.String(); strings.Contains(page, "/search?type=") {
			t.Errorf("Search from %s is everything", from)
		}
	}
	body := get(t, h, "/search?type=task").Body.String()
	if !strings.Contains(body, "<h1>Search tasks</h1>") || !strings.Contains(body, `Searching tasks only. <a class="sw-link" href="/search">Search everything</a>`) {
		t.Errorf("the search of tasks says so, and offers everything\n%s", truncate(body))
	}
	if strings.Index(body, "Search everything") > strings.Index(body, `<form method="get"`) {
		t.Error("the way to everything comes before the box")
	}
}

// Paging keeps the kind, and the kinds keep the words and start at page 1.
func TestSearchPagesKeepTheKind(t *testing.T) {
	_, h := newApp(t)
	for i := range 25 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": fmt.Sprintf("Seeds %d", i)}), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Seeds note"}), http.StatusCreated)
	body := get(t, h, "/search?q=seeds&type=task").Body.String()
	if !strings.Contains(body, `href="/search?page=2&amp;q=seeds&amp;type=task"`) {
		t.Errorf("the next page keeps the kind\n%s", truncate(body))
	}
	two := get(t, h, "/search?q=seeds&type=task&page=2").Body.String()
	if !strings.Contains(two, "<title>Search: seeds in tasks, 25 results, page 2 of 2") || strings.Contains(two, "Seeds note") {
		t.Errorf("page 2 is still tasks\n%s", truncate(two))
	}
	if !strings.Contains(two, `href="/search?q=seeds">All (26)</a>`) || strings.Contains(two, "page=2&amp;q=seeds&amp;type=note") {
		t.Error("the kinds keep the words and go to the first page")
	}
}

// A kind there is none of, or one the system keeps, is refused plainly.
func TestSearchRefusesAKindItCannotNarrowTo(t *testing.T) {
	h := seeds(t)
	for _, kind := range []string{"zebra", "message", "block", "activity"} {
		page := get(t, h, "/search?q=seeds&type="+kind)
		wantStatus(t, page, http.StatusBadRequest)
		body := page.Body.String()
		if !strings.Contains(body, "There is no kind of thing called “"+kind+"” to search in.") || !strings.Contains(body, `href="/search?q=seeds">Search everything</a>`) {
			t.Errorf("%s is refused in plain words, with everything offered\n%s", kind, truncate(body))
		}
		if strings.Contains(body, "Buy seeds") {
			t.Errorf("%s shows no results", kind)
		}
	}
}
