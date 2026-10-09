package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// kindsMonth is September 2026 with two tasks and a reminder, and a task
// in October, and a calendar of everything; it says the block's id.
func kindsMonth(t *testing.T, h http.Handler, props map[string]any) string {
	t.Helper()
	for _, task := range []map[string]any{{"title": "Order compost", "due": "2026-09-19T00:00:00Z"}, {"title": "Plant garlic", "due": "2026-09-22T00:00:00Z"}, {"title": "Harvest", "due": "2026-10-03T00:00:00Z"}} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", task), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{"title": "Call the vet", "at": "2026-09-10T00:00:00Z"}), http.StatusCreated)
	return addCalendar(t, h, props)
}

func addCalendar(t *testing.T, h http.Handler, props map[string]any) string {
	t.Helper()
	var block struct{ ID string }
	rec := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "calendar", "props": props})
	wantStatus(t, rec, http.StatusCreated)
	decode(t, rec, &block)
	return block.ID
}

// A month of everything with more than one kind offers each kind as a
// link with its count in the month, All first and shown; a link narrows.
func TestACalendarOfEverythingIsNarrowedByKind(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	id := kindsMonth(t, h, map[string]any{"type": "all", "month": "2026-09", "detail": "page"})
	own := "/canvas/" + id
	page := get(t, h, own).Body.String()
	for _, want := range []string{
		`<nav class="sw-filters sw-filters--links" data-component="filters" data-shape="links" aria-label="Kinds of event, September 2026">`,
		`href="` + own + `" aria-current="page">All (3)</a>`,
		`href="` + own + `?c-` + id + `-type=reminder">Reminders (1)</a>`,
		`href="` + own + `?c-` + id + `-type=task">Tasks (2)</a>`,
		"Order compost", "Call the vet",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the kinds should be offered: %s\n%s", want, truncate(page))
		}
	}
	tasks := get(t, h, own+"?c-"+id+"-type=task").Body.String()
	for _, want := range []string{
		`?c-` + id + `-type=task" aria-current="page">Tasks (2)</a>`, `href="` + own + `">All (3)</a>`,
		">Order compost</a>", ">Plant garlic</a>",
	} {
		if !strings.Contains(tasks, want) {
			t.Errorf("narrowed to tasks: %s", want)
		}
	}
	if strings.Contains(tasks, "Call the vet") {
		t.Error("a reminder is not a task")
	}
	// The months either side and each day keep the kind.
	for _, want := range []string{
		`href="` + own + `?c-` + id + `-type=task&amp;month=2026-08" rel="prev"`,
		`href="` + own + `?c-` + id + `-type=task&amp;month=2026-10" rel="next"`,
		`?c-` + id + `-type=task&amp;day=2026-09-19`,
	} {
		if !strings.Contains(tasks, want) {
			t.Errorf("moving through time keeps the kind: %s\n%s", want, truncate(tasks))
		}
	}
	next := get(t, h, own+"?c-"+id+"-type=task&month=2026-10").Body.String()
	if !strings.Contains(next, ">Harvest</a>") || !strings.Contains(next, `aria-current="page">Tasks (1)</a>`) || !strings.Contains(next, `href="`+own+`?month=2026-10">All (1)</a>`) {
		t.Errorf("next month keeps the kind, counts its own and All keeps the month\n%s", truncate(next))
	}
	if bogus := get(t, h, own+"?c-"+id+"-type=spaceship").Body.String(); !strings.Contains(bogus, `aria-current="page">All (3)</a>`) || !strings.Contains(bogus, "Call the vet") {
		t.Error("a kind the calendar does not have narrows nothing")
	}
}

// A calendar of one type is never offered another, nor widened by the
// address; one kind in the month needs no choice.
func TestACalendarsKindNeverWidens(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	id := kindsMonth(t, h, map[string]any{"type": "task", "where": []string{"title~garlic"}, "month": "2026-09", "detail": "page"})
	page := get(t, h, "/canvas/"+id+"?c-"+id+"-type=reminder").Body.String()
	if strings.Contains(page, "sw-filters") || strings.Contains(page, "Call the vet") || strings.Contains(page, "Order compost") || !strings.Contains(page, "Plant garlic") {
		t.Errorf("a calendar of tasks keeps to its own where\n%s", truncate(page))
	}
	only := addCalendar(t, h, map[string]any{"type": "all", "month": "2026-10", "detail": "page"})
	if page := get(t, h, "/canvas/"+only).Body.String(); strings.Contains(page, "sw-filters") {
		t.Error("one kind in the month offers no choice")
	}
}

// Two calendars on a canvas keep their own kind, each link comes back to
// its block, and the page's other fields are kept.
func TestCalendarKindsArePerBlock(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	first := kindsMonth(t, h, map[string]any{"type": "all", "month": "2026-09"})
	second := addCalendar(t, h, map[string]any{"type": "all", "month": "2026-09"})
	page := get(t, h, "/?c-"+first+"-type=reminder").Body.String()
	a, b := strings.Index(page, `id="block-`+first+`"`), strings.Index(page, `id="block-`+second+`"`)
	if a < 0 || b < 0 {
		t.Fatalf("both calendars on the canvas\n%s", truncate(page))
	}
	one, two := page[a:], page[b:]
	if a < b {
		one = page[a:b]
	} else {
		two = page[b:a]
	}
	if !strings.Contains(one, `aria-current="page">Reminders (1)</a>`) || strings.Contains(one, "Order compost") {
		t.Errorf("the first is narrowed to reminders\n%s", one)
	}
	if !strings.Contains(two, `aria-current="page">All (3)</a>`) || !strings.Contains(two, "Order compost") {
		t.Error("the second keeps all of them")
	}
	link := two[strings.Index(two, ">Tasks (2)<")-200:]
	link = link[strings.LastIndex(link[:200], `href="`):]
	for _, want := range []string{"c-" + first + "-type=reminder", "c-" + second + "-type=task", "#block-" + second + `"`} {
		if !strings.Contains(link[:strings.Index(link, ">")], want) {
			t.Errorf("a link keeps the other calendar's kind and comes back to its own block: %s in %s", want, link)
		}
	}
}
