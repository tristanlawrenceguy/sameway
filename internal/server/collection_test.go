package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A collection block is the records that match, kept current: the same
// query on the canvas, on the list page and over the API, and a wrong
// condition explained rather than rendered as nothing.
func TestACollectionIsTheRecordsThatMatch(t *testing.T) {
	a, h := newApp(t)
	day := func(d int) string { return time.Now().AddDate(0, 0, d).UTC().Format(time.RFC3339) }
	for _, task := range []map[string]any{
		{"title": "Dig the pond", "due": day(-2)},
		{"title": "Order compost", "due": day(2)},
		{"title": "Plant garlic", "due": day(5)},
		{"title": "Call the dentist", "due": day(3), "done": true},
		{"title": "Read the seed catalogue", "due": day(30)},
	} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", task), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "where": []string{"done=false", "due<=+7d", "due>=today"}, "order": "due", "label": "Due this week"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()
	for _, want := range []string{`data-component="collection"`, "Due this week", "Order compost", "Plant garlic", `href="/t/task?order=due&amp;where=done%3Dfalse&amp;where=due%3C%3D%2B7d&amp;where=due%3E%3Dtoday"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the canvas should show the matching tasks with %s", want)
		}
	}
	for _, not := range []string{"Dig the pond", "Call the dentist", "seed catalogue"} {
		if strings.Contains(page, not) {
			t.Errorf("%s does not match and should not show", not)
		}
	}
	if strings.Index(page, "Order compost") > strings.Index(page, "Plant garlic") {
		t.Error("the tasks come in due order")
	}
	if !strings.Contains(page, "Due "+when.Text(day(2))) {
		t.Error("each task shows the day it is due")
	}

	// The list page takes the same query and says what it is showing.
	list := get(t, h, "/t/task?"+url.Values{"where": {"done=false", "due<today"}}.Encode())
	wantStatus(t, list, http.StatusOK)
	if body := list.Body.String(); !strings.Contains(body, "Dig the pond") || strings.Contains(body, "Order compost") || !strings.Contains(body, "1 matching done=false, due&lt;today") {
		t.Errorf("the list page filters and says so: %.500s", body)
	}
	bad := get(t, h, "/t/task?where=owner%3Dme")
	wantStatus(t, bad, http.StatusBadRequest)
	if !strings.Contains(bad.Body.String(), `no field &#34;owner&#34;`) || !strings.Contains(bad.Body.String(), "See all tasks") {
		t.Errorf("a wrong condition says what the type has and how to see everything: %.500s", bad.Body.String())
	}

	// The API takes it too.
	var out struct {
		Count   int
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/task?where=done%3Dtrue"), &out)
	if out.Count != 1 || out.Records[0].Fields["title"] != "Call the dentist" {
		t.Errorf("the API filters with the same words, got %+v", out)
	}
	wantStatus(t, get(t, h, "/api/task?where=owner%3Dme"), http.StatusBadRequest)

	// A wrong condition on the canvas is said in words, not rendered as nothing.
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "where": []string{"owner=me"}, "label": "Mine"},
	}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `no field &#34;owner&#34;; it has title, done, due`) {
		t.Errorf("the block should explain the wrong field: %.300s", page[strings.Index(page, "Mine"):])
	}
	_ = a
}

// The same matches can be a table with a column per chosen field, or
// cards with the fields under each title; a ref shows the title it
// points at and a date its day.
func TestACollectionCanBeATableOrCards(t *testing.T) {
	_, h := newApp(t)
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost", "due": "2026-10-02T00:00:00Z", "project": garden.ID}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "as": "table", "show": []string{"due", "project", "done"}, "label": "Open tasks"},
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "project", "as": "cards", "show": []string{"status"}, "label": "Projects"},
	}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	for _, want := range []string{
		`<th scope="col">Title</th><th scope="col">Due</th><th scope="col">Project</th><th scope="col">Done</th>`,
		`<th scope="row"><a class="sw-link" href="/t/task/`, `<td>Fri 2 Oct 2026</td><td>Garden</td><td>no</td>`,
		`sw-collection__cards`, `<dt>Status</dt><dd>active</dd>`, `href="/t/project/` + garden.ID + `">Garden</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the canvas should carry %s", want)
		}
	}
}
