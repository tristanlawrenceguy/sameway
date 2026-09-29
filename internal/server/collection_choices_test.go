package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// seedTasks makes eight tasks: six not done, due from two days ago to a
// month on, and two done.
func seedTasks(t *testing.T, a *app.App) {
	t.Helper()
	day := func(d int) string { return time.Now().AddDate(0, 0, d).UTC().Format(time.RFC3339) }
	for _, task := range []map[string]any{
		{"title": "Dig the pond", "due": day(-2)},
		{"title": "Order compost", "due": day(2)},
		{"title": "Plant garlic", "due": day(5)},
		{"title": "Buy seeds", "due": day(30)},
		{"title": "Empty the water butt"},
		{"title": "Mend the fence", "due": day(-1)},
		{"title": "Call the dentist", "due": day(-3), "done": true},
		{"title": "Wash the car", "due": day(3), "done": true},
	} {
		if _, err := a.Store.Create("task", task); err != nil {
			t.Fatal(err)
		}
	}
}

// addCollection puts a collection on the canvas and says its block's id.
func addCollection(t *testing.T, h http.Handler, props map[string]any) string {
	t.Helper()
	rec := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": props})
	wantStatus(t, rec, http.StatusCreated)
	var out map[string]any
	decode(t, rec, &out)
	id, _ := out["id"].(string)
	if id == "" {
		t.Fatalf("no block id: %s", rec.Body.String())
	}
	return id
}

// section is one collection block's part of a page.
func section(t *testing.T, page, id string) string {
	t.Helper()
	start := strings.Index(page, `id="collection-`+id+`"`)
	if start < 0 {
		t.Fatalf("no collection %s on the page", id)
	}
	end := strings.Index(page[start:], "</section>")
	return page[start : start+end]
}

// inOrder says whether the titles come in this order in s.
func inOrder(s string, titles ...string) bool {
	at := -1
	for _, title := range titles {
		i := strings.Index(s, ">"+title+"<")
		if i < at {
			return false
		}
		at = i
	}
	return true
}

// A person narrows and sorts a collection where it is, with choices the
// type's fields make sense for, in a GET form that needs no script.
func TestACollectionOffersChoicesFromItsSchema(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks"})
	list := section(t, get(t, h, "/").Body.String(), id)
	for _, want := range []string{
		`<form method="get" action="/#collection-` + id + `"><fieldset class="sw-filters__form"><legend class="sw-visually-hidden">Show and sort: Tasks</legend>`,
		`name="c-` + id + `-sort"`, `<option value="-created_at" selected>Newest first</option>`,
		`>A to Z<`, `<option value="due">Due soonest first</option>`,
		`name="c-` + id + `-done"`, `>Not done<`,
		`name="c-` + id + `-due"`, `>Due before today<`, `>Due in the next 7 days<`,
		`<button type="submit" class="sw-button sw-button--secondary sw-pressable">Apply`,
		`8 tasks.`,
	} {
		if !strings.Contains(list, want) {
			t.Errorf("the choices should have %s: %.3000s", want, list)
		}
	}
	if strings.Contains(list, "Showing:") || strings.Contains(list, "Reset") {
		t.Error("as it was set up, nothing is said to be chosen and there is nothing to reset")
	}

	// A project has a pick-list, and no yes-or-no or date of its own.
	if _, err := a.Store.Create("project", map[string]any{"title": "Garden"}); err != nil {
		t.Fatal(err)
	}
	p := addCollection(t, h, map[string]any{"type": "project", "controls": true})
	projects := section(t, get(t, h, "/").Body.String(), p)
	if !strings.Contains(projects, `name="c-`+p+`-status"`) || !strings.Contains(projects, `>Any<`) || !strings.Contains(projects, `>Active<`) {
		t.Errorf("a project is narrowed by its status: %.2000s", projects)
	}
	if strings.Contains(projects, `-done"`) || strings.Contains(projects, "soonest") {
		t.Error("only what the schema has is offered")
	}
}

// The person's choices add to the assistant's where and never undo it; a
// value the controls do not offer is ignored, so the address cannot widen
// or add a condition of its own.
func TestChoicesNarrowWithinWhereAndSort(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "where": []string{"done=false"}, "label": "Open", "controls": true})
	p := "c-" + id + "-"

	base := section(t, get(t, h, "/").Body.String(), id)
	if strings.Contains(base, `name="`+p+`done"`) {
		t.Error("done is fixed by where, so it is not offered")
	}
	wide := section(t, get(t, h, "/?"+url.Values{p + "done": {"true"}, p + "sort": {"bogus"}, p + "due": {"due>0"}}.Encode()).Body.String(), id)
	if strings.Contains(wide, "Call the dentist") || strings.Contains(wide, "Wash the car") || !strings.Contains(wide, "Dig the pond") {
		t.Error("the address cannot show what where leaves out")
	}

	past := section(t, get(t, h, "/?"+url.Values{p + "due": {"past"}, p + "sort": {"due"}}.Encode()).Body.String(), id)
	for _, want := range []string{"2 tasks.", "Showing: due before today, due soonest first.", `<option value="past" selected>`, `<option value="due" selected>`} {
		if !strings.Contains(past, want) {
			t.Errorf("narrowed, it says %s: %.3000s", want, past)
		}
	}
	if !inOrder(past, "Dig the pond", "Mend the fence") {
		t.Error("the older due day comes first")
	}
	if strings.Contains(past, "Order compost") || strings.Contains(past, "Call the dentist") {
		t.Error("due before today and still not done: nothing else")
	}

	az := section(t, get(t, h, "/?"+p+"sort=title").Body.String(), id)
	if !inOrder(az, "Buy seeds", "Dig the pond", "Empty the water butt", "Mend the fence", "Order compost", "Plant garlic") {
		t.Errorf("A to Z: %.3000s", az)
	}
	soon := section(t, get(t, h, "/?"+p+"sort=due").Body.String(), id)
	if !inOrder(soon, "Dig the pond", "Mend the fence", "Order compost", "Plant garlic", "Buy seeds", "Empty the water butt") {
		t.Error("soonest first, and one with no day comes last")
	}
}

// Two collections on one canvas keep their own choices; each form keeps
// the other's and the page's, and Reset takes away only its own.
func TestChoicesArePerBlock(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	one := addCollection(t, h, map[string]any{"type": "task", "label": "One", "controls": true})
	two := addCollection(t, h, map[string]any{"type": "task", "label": "Two", "controls": true})
	page := get(t, h, "/?"+url.Values{"c-" + one + "-done": {"true"}, "prompt": {"hello"}}.Encode()).Body.String()
	first, second := section(t, page, one), section(t, page, two)
	if strings.Contains(first, "Dig the pond") || !strings.Contains(first, "Wash the car") {
		t.Error("the first shows done tasks only")
	}
	if !strings.Contains(second, "Dig the pond") || strings.Contains(second, "Showing:") {
		t.Error("the second is as it was set up")
	}
	if !strings.Contains(second, `<input type="hidden" name="c-`+one+`-done" value="true">`) || !strings.Contains(second, `<input type="hidden" name="prompt" value="hello">`) {
		t.Errorf("the second's form keeps the first's choice and the page's own: %.2000s", second)
	}
	if !strings.Contains(first, `href="/?prompt=hello#collection-`+one+`">Reset`) {
		t.Errorf("Reset takes away only this list's choices: %.3000s", first)
	}
	if strings.Contains(first, `<input type="hidden" name="c-`+one) {
		t.Error("a form does not carry its own choices twice")
	}
}

// The list page link carries the choices, a list cut short says so, and
// nothing matching says so with Reset beside it.
func TestChoicesKeepToTheListPageAndSayWhenNothingMatches(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks", "limit": 6})
	p := "c-" + id + "-"
	cut := section(t, get(t, h, "/?"+url.Values{p + "done": {"false"}, p + "sort": {"title"}}.Encode()).Body.String(), id)
	if !strings.Contains(cut, "6 tasks.") || strings.Contains(cut, "Showing the first") {
		t.Errorf("six match and all six are shown: %.3000s", cut)
	}
	if !strings.Contains(cut, `href="/t/task?order=title&amp;where=done%3Dfalse"`) {
		t.Errorf("the list page has the same choices: %.3000s", cut)
	}
	sorted := section(t, get(t, h, "/?"+p+"sort=title").Body.String(), id)
	if !strings.Contains(sorted, "8 tasks.") || !strings.Contains(sorted, "Showing the first 6.") {
		t.Errorf("more match than are shown, and it says so: %.3000s", sorted)
	}
	none := section(t, get(t, h, "/?"+url.Values{p + "done": {"true"}, p + "due": {"week"}}.Encode()).Body.String(), id)
	if !strings.Contains(none, "Wash the car") {
		t.Fatalf("done and due this week: %.3000s", none)
	}
	none = section(t, get(t, h, "/?"+url.Values{p + "done": {"true"}, p + "due": {"past"}, p + "sort": {"due"}}.Encode()).Body.String(), id)
	if !strings.Contains(none, "Call the dentist") {
		t.Fatalf("done and due before today: %.3000s", none)
	}
	if _, err := a.Store.Create("project", map[string]any{"title": "Garden", "status": "active"}); err != nil {
		t.Fatal(err)
	}
	pr := addCollection(t, h, map[string]any{"type": "project", "controls": true})
	empty := section(t, get(t, h, "/?c-"+pr+"-status=done").Body.String(), pr)
	if !strings.Contains(empty, "Nothing matches:") || !strings.Contains(empty, ">Reset<") {
		t.Errorf("nothing matching says so, with Reset: %.2000s", empty)
	}
}

// The choices are on where they help: not for a short list, not when the
// block turns them off, always on the block's own page.
func TestChoicesAreOfferedWhereTheyHelp(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	few := addCollection(t, h, map[string]any{"type": "task", "where": []string{"done=true"}, "label": "Done"})
	off := addCollection(t, h, map[string]any{"type": "task", "label": "All", "controls": false, "choices": map[string]any{"action": "/elsewhere", "showing": "made up"}})
	short := addCollection(t, h, map[string]any{"type": "task", "label": "Top three", "limit": 3})
	many := addCollection(t, h, map[string]any{"type": "task", "label": "Many"})
	page := get(t, h, "/").Body.String()
	for _, id := range []string{few, off, short} {
		if strings.Contains(section(t, page, id), "sw-filters__form") {
			t.Errorf("collection %s should not offer choices, nor keep ones it was given", id)
		}
	}
	if !strings.Contains(section(t, page, many), "sw-filters__form") {
		t.Error("eight tasks are worth narrowing")
	}
	own := get(t, h, "/canvas/"+few).Body.String()
	if !strings.Contains(own, `action="/canvas/`+few+`#collection-`+few+`"`) {
		t.Errorf("the block's own page offers them, and comes back to itself: %.500s", own)
	}
	kept := get(t, h, "/?c-"+short+"-sort=title").Body.String()
	if !strings.Contains(section(t, kept, short), ">Reset<") {
		t.Error("a choice already made keeps its way back")
	}
}
