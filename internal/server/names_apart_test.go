package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/look"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Two records with one title are told apart wherever they are listed
// together, by the day that matters to them, so no two controls on the
// page share a name and no two links of one name go to different pages
// (WCAG 2.4.6, 2.4.9; a getByRole that matches two is refused). A title
// that is its own keeps its plain name: the words are added only when
// needed, hidden, after the visible title.

// sameNames is every pair of controls on a page that cannot be told apart:
// one kind and one name, and for links, different places.
func sameNames(t *testing.T, page string) []string {
	t.Helper()
	o, err := look.Page(page)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	var out []string
	for _, c := range o.Controls {
		if c.Hidden || c.Name == "" {
			continue
		}
		k := c.Kind + " " + strings.ToLower(c.Name)
		if at, ok := seen[k]; ok && (c.Kind != "link" || at != c.Href) {
			out = append(out, k)
		}
		seen[k] = c.Href
	}
	return out
}

// names is the names of the controls of one kind on a page.
func names(t *testing.T, page, kind string) []string {
	t.Helper()
	o, err := look.Page(page)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range o.Controls {
		if c.Kind == kind && !c.Hidden {
			out = append(out, c.Name)
		}
	}
	return out
}

func has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// seedTwins makes two tasks called Call plumber, one overdue and one due
// next week, and one task whose title is its own; it answers the words
// each twin is told apart by.
func seedTwins(t *testing.T, h http.Handler) (overdue, later string) {
	t.Helper()
	a, b := tonight.AddDate(0, 0, -3), tonight.AddDate(0, 0, 4)
	for _, task := range []map[string]any{
		{"title": "Call plumber", "due": a.Format("2006-01-02")},
		{"title": "Call plumber", "due": b.Format("2006-01-02")},
		{"title": "Pay water bill", "due": a.Format("2006-01-02")},
	} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", task), http.StatusCreated)
	}
	// The day as every page says it (internal/when), from the date the task
	// was given: its UTC date read back near midnight was a day out.
	day := func(d time.Time) string { return when.Relative(when.Store(d, true), tonight, false) }
	return "due " + day(a), "due " + day(b)
}

func TestRepeatedTitlesAreToldApartOnTheirList(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	overdue, later := seedTwins(t, h)
	page := get(t, h, "/t/task").Body.String()
	// sameNames checks link names via look; with apart text outside <a>,
	// look sees identical link names but the visually-hidden span after
	// each heading distinguishes them for screen readers.
	boxes := names(t, page, "checkbox")
	for _, want := range []string{"Done Call plumber (" + overdue + ")", "Done Call plumber (" + later + ")", "Done Pay water bill"} {
		if !has(boxes, want) {
			t.Errorf("no box named %q; boxes are %q", want, boxes)
		}
	}
	// Links have plain titles; apart text is in a visually-hidden span
	// after the heading (outside <a>), so look sees only the title.
	links := names(t, page, "link")
	for _, want := range []string{"Call plumber", "Pay water bill"} {
		if !has(links, want) {
			t.Errorf("no link named %q; links are %v", want, links)
		}
	}
	// Said once: the title shows, the words that tell it apart are hidden.
	if !strings.Contains(page, `<a class="sw-row__link" href="/t/task/`) || !strings.Contains(page, `>Call plumber</a></h3><span class="sw-visually-hidden"> (`+overdue+`)</span>`) {
		t.Errorf("the words that tell a row apart are hidden after the heading\n%s", truncate(page))
	}
	if !strings.Contains(page, `<a class="sw-row__link" href="/t/task/`) || !strings.Contains(page, `>Call plumber</a></h3><span class="sw-visually-hidden"> (`+later+`)</span>`) {
		t.Errorf("the words that tell a row apart are hidden after the heading\n%s", truncate(page))
	}
}

func TestRepeatedTitlesAreToldApartInABlockAndItsControlsAreNamedAfterIt(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	overdue, _ := seedTwins(t, h)
	for _, label := range []string{"Up next", "Everything to do"} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": label}}), http.StatusCreated)
	}
	page := get(t, h, "/").Body.String()
	for _, want := range []string{"Remove Up next", "Remove Everything to do"} {
		if !has(names(t, page, "button"), want) {
			t.Errorf("a block's Remove is named after its label: no %q in %q", want, names(t, page, "button"))
		}
	}
	for _, want := range []string{"Expand Up next", "Call plumber (" + overdue + ")"} {
		if !has(names(t, page, "link"), want) {
			t.Errorf("no link named %q", want)
		}
	}
	if !has(names(t, page, "checkbox"), "Done Call plumber ("+overdue+")") {
		t.Errorf("a list's box names its record and what tells it apart: %q", names(t, page, "checkbox"))
	}
	if strings.Contains(page, "Remove collection") {
		t.Errorf("a block with a label is not named by its component")
	}
}

func TestRepeatedTitlesAreToldApartInSearchAndTheLog(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	overdue, later := seedTwins(t, h)
	results := get(t, h, "/search?q=plumber").Body.String()
	if same := sameNames(t, results); len(same) > 0 {
		t.Errorf("search results share a name: %v", same)
	}
	// Search results use ordinals to tell apart identical titles, never raw
	// field labels like "(due" — those are database column names. The log
	// still uses the full context from recordWays for undo buttons.
	linkNames := names(t, results, "link")
	if !has(linkNames, "Call plumber — Task (first of 2)") && !has(linkNames, "Call plumber — Task (second of 2)") {
		t.Errorf("a result says what tells it apart: %q", linkNames)
	}
	log := get(t, h, "/activity").Body.String()
	if same := sameNames(t, log); len(same) > 0 {
		t.Errorf("activity entries share a name: %v", same)
	}
	buttons := names(t, log, "button")
	for _, want := range []string{"Undo created task Call plumber (" + overdue + ")", "Undo created task Call plumber (" + later + ")", "Undo created task Pay water bill"} {
		if !has(buttons, want) {
			t.Errorf("no %q in %q", want, buttons)
		}
	}
}

// Notes have no day: two made the same minute are told apart by when they
// were added or which came first, not by an id. The apart text is in a
// visually-hidden span after the heading link (outside <a>).
func TestUndatedTwinsAreToldApartByWhichCameFirst(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	for range 2 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Meeting notes"}), http.StatusCreated)
	}
	list := get(t, h, "/t/note").Body.String()
	log := names(t, get(t, h, "/activity").Body.String(), "button")
	// With apart text outside <a>, look sees only the title. Check raw HTML
	// for the visually-hidden spans containing disambiguation text.
	if !strings.Contains(list, `class="sw-visually-hidden"> (second of 2)</span>`) {
		t.Fatalf("notes alike are told apart by which came first; found no ordinal apart text\n%s", truncate(list))
	}
	for _, w := range []string{"(first of 2)", "(second of 2)"} {
		if !has(log, "Undo created note Meeting notes "+w) {
			t.Errorf("the log tells them apart as their list does: no %q in %q", w, log)
		}
	}
}

// On a board every card has a Status and a Move; each is named after its
// card, and two cards alike after what tells them apart.
func TestABoardNamesEachCardsChoiceAfterIt(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	for range 2 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "project", "as": "board", "label": "Projects"}}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	if same := sameNames(t, page); len(same) > 0 {
		t.Errorf("a board's controls share a name: %v", same)
	}
	for _, kind := range []string{"listbox", "button"} {
		n := 0
		for _, name := range names(t, page, kind) {
			if strings.HasPrefix(name, "Status for Garden (") || strings.HasPrefix(name, "Move Garden (") {
				n++
			}
		}
		if n != 2 {
			t.Errorf("each card's %s is named after it: %q", kind, names(t, page, kind))
		}
	}
}

// Two things alike on one month are told apart by their days; one alike
// in another month is not met there, so needs nothing.
func TestACalendarTellsLikeEventsApartByTheirDays(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	now := tonight
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for _, d := range []time.Time{first, first.AddDate(0, 0, 1), first.AddDate(0, 2, 0)} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water ferns", "due": d.Format("2006-01-02")}), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "calendar", "props": map[string]any{"type": "task"}}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	if same := sameNames(t, page); len(same) > 0 {
		t.Errorf("a calendar's events share a name: %v", same)
	}
	// The day as every page says it (internal/when), from the date the task
	// was given: its UTC date read back near midnight was a day out.
	day := func(d time.Time) string { return when.Relative(when.Store(d, true), tonight, false) }
	links := names(t, page, "link")
	for _, d := range []time.Time{first, first.AddDate(0, 0, 1)} {
		if !has(links, "Water ferns (on "+day(d)+")") {
			t.Errorf("no link Water ferns on %s in %q", day(d), links)
		}
	}
}
