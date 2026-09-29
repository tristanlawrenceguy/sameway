package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// storedBlock puts a block straight into the store, as one written before
// blocks were checked when written, and returns its id.
func storedBlock(t *testing.T, a *app.App, component string, props map[string]any) string {
	t.Helper()
	rec, err := a.Store.Create("block", map[string]any{"component": component, "props": props, "position": 0, "created_by": "assistant", "actor": "assistant"})
	if err != nil {
		t.Fatal(err)
	}
	return rec.ID
}

// call runs one of the assistant's tools as the model would.
func call(t *testing.T, a *app.App, tool string, args map[string]any) (string, bool) {
	t.Helper()
	raw, _ := json.Marshal(args)
	return a.Chat.Call(tool, raw)
}

func blockCount(t *testing.T, a *app.App) int {
	t.Helper()
	blocks, err := a.Store.List("block", store.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return len(blocks)
}

// The blocks the evaluation's models were told were added, each of which
// only said on the page that it was set up wrong.
var setUpWrong = []struct {
	component string
	props     map[string]any
	why       string
}{
	{"collection", map[string]any{"type": "entry", "as": "board"}, "a board needs a pick-list field for its columns, and entry has none; show it as a list, table or cards, or add a pick-list field first"},
	{"collection", map[string]any{"type": "task", "where": []any{"owner=me"}}, `task has no field "owner"; it has title, done, status, due`},
	{"chart", map[string]any{"type": "entry", "by": "at", "period": "day", "sum": "glasses"}, `sum needs a number field on entry; "glasses" is not one; its number fields are amount`},
	{"chart", map[string]any{"type": "task", "by": "owner"}, `task has no field "owner" to group by; it has title, done`},
	{"calendar", map[string]any{"type": "tasks"}, "there is no content type tasks (did you mean task?); the workspace has"},
	{"calendar", map[string]any{"type": "note"}, "note has no date field to place on a calendar; show it as a list instead"},
	{"tracker", map[string]any{"tags": []any{"garden"}}, "no habit is tagged garden; the habits have health"},
}

func withHabit(t *testing.T, a *app.App) {
	t.Helper()
	if _, err := a.Store.Create("habit", map[string]any{"name": "Drink water", "tags": []any{"health"}, "unit": "glasses"}); err != nil {
		t.Fatal(err)
	}
}

// add_component and update_component check a block the way its page will
// resolve it: one that could only say it is set up wrong is refused with
// the page's own reason and what to do, and nothing is written.
func TestTheAssistantsBlockToolsRefuseWhatCouldNotBeShown(t *testing.T) {
	a, _ := newApp(t)
	withHabit(t, a)
	before := blockCount(t, a)
	for _, c := range setUpWrong {
		said, isErr := call(t, a, "add_component", map[string]any{"component": c.component, "props": c.props})
		if !isErr || !strings.Contains(said, "not saved: this "+c.component+" could not be shown: "+c.why) || !strings.Contains(said, "call add_component again") {
			t.Errorf("add_component %s %v should be refused with %q, got %v %q", c.component, c.props, c.why, isErr, said)
		}
	}
	if n := blockCount(t, a); n != before {
		t.Errorf("a refused block is not added: %d blocks, were %d", n, before)
	}

	said, isErr := call(t, a, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}})
	if isErr {
		t.Fatal(said)
	}
	id := strings.Fields(strings.SplitAfter(said, "as block ")[1])[0]
	for _, c := range setUpWrong {
		if c.component != "collection" {
			continue
		}
		said, isErr := call(t, a, "update_component", map[string]any{"id": id, "props": c.props})
		if !isErr || !strings.Contains(said, c.why) || !strings.Contains(said, "call update_component again") {
			t.Errorf("update_component to %v should be refused with %q, got %q", c.props, c.why, said)
		}
	}
	rec, _ := a.Store.Get("block", id)
	if props, _ := rec.Fields["props"].(map[string]any); props["as"] != nil || props["where"] != nil {
		t.Errorf("a refused change leaves the block as it was: %v", props)
	}

	// Asked first, the question is not put: the person's Yes would only
	// find it refused.
	said, isErr = call(t, a, "propose_change", map[string]any{"summary": "Add a board?", "tool": "add_component", "component": "collection", "props": setUpWrong[0].props})
	if !isErr || !strings.Contains(said, "nothing asked") || !strings.Contains(said, setUpWrong[0].why) {
		t.Errorf("a question carrying a block that could not be shown is not asked, got %q", said)
	}
	if len(a.Chat.Proposals()) != 0 {
		t.Error("no question is left waiting")
	}

	// An arrangement is checked whole, so none of it is added.
	before = blockCount(t, a)
	said, isErr = call(t, a, "add_arrangement", map[string]any{"name": "week", "fills": map[string]any{"calendar": map[string]any{"type": "tasks"}}})
	if !isErr || !strings.Contains(said, "block calendar") || !strings.Contains(said, "did you mean task?") {
		t.Errorf("an arrangement with a block that could not be shown is refused, got %q", said)
	}
	if n := blockCount(t, a); n != before {
		t.Errorf("no half of an arrangement is left behind: %d blocks, were %d", n, before)
	}
}

// Through the API, the same blocks are 422 with the same words, and none
// is added; a block moved without new props is left alone, as is one
// stored before blocks were checked, which still says what is wrong.
func TestTheAPIRefusesABlockThatCouldNotBeShown(t *testing.T) {
	a, h := newApp(t)
	withHabit(t, a)
	before := blockCount(t, a)
	for _, c := range setUpWrong {
		res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": c.component, "props": c.props})
		wantStatus(t, res, http.StatusUnprocessableEntity)
		if body := res.Body.String(); !strings.Contains(body, `"cannot_show"`) || !strings.Contains(body, jsonText(c.why)) {
			t.Errorf("POST %s %v should say %q: %s", c.component, c.props, c.why, body)
		}
	}
	if n := blockCount(t, a); n != before {
		t.Errorf("a refused block is not added: %d blocks, were %d", n, before)
	}

	var made struct {
		ID    string
		Shows string
	}
	res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "tracker", "props": map[string]any{"tags": []any{"health"}}})
	wantStatus(t, res, http.StatusCreated)
	decode(t, res, &made)
	if made.Shows != "1 habit tagged health" {
		t.Errorf("a block written says what it shows, got %q", made.Shows)
	}
	res = postJSON(t, h, http.MethodPatch, "/api/block/"+made.ID, map[string]any{"props": map[string]any{"tags": []any{"garden"}}})
	wantStatus(t, res, http.StatusUnprocessableEntity)
	if !strings.Contains(res.Body.String(), "no habit is tagged garden") {
		t.Errorf("PATCH to props that could not be shown says why: %s", res.Body.String())
	}

	broken := storedBlock(t, a, "collection", setUpWrong[0].props)
	wantStatus(t, postJSON(t, h, http.MethodPatch, "/api/block/"+broken, map[string]any{"span": 6}), http.StatusOK)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "This list cannot be shown") {
		t.Error("a block stored before the check still renders what is wrong")
	}
}

// jsonText is words as they read inside a JSON string.
func jsonText(s string) string {
	raw, _ := json.Marshal(s)
	return strings.Trim(string(raw), `"`)
}

// A block written says what it shows, so a model has something to notice:
// how many, which and in what order; what a chart draws and over what.
func TestABlockWrittenSaysWhatItShows(t *testing.T) {
	a, _ := newApp(t)
	withHabit(t, a)
	for _, task := range []map[string]any{{"title": "a", "due": "2026-10-01T00:00:00Z"}, {"title": "b"}, {"title": "c"}, {"title": "d", "done": true}} {
		if _, err := a.Store.Create("task", task); err != nil {
			t.Fatal(err)
		}
	}
	habits, _ := a.Store.List("habit", store.ListOptions{})
	for _, day := range []string{"2026-09-01T08:00:00Z", "2026-09-01T12:00:00Z", "2026-09-02T08:00:00Z"} {
		if _, err := a.Store.Create("entry", map[string]any{"habit": habits[0].ID, "at": day, "amount": 2}); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		component string
		props     map[string]any
		shows     string
	}{
		{"collection", map[string]any{"type": "task", "where": []any{"done=false"}, "order": "due", "label": "Up next"}, "; it shows 3 tasks, not done, by due"},
		{"collection", map[string]any{"type": "task", "as": "table", "label": "All"}, "; it shows 4 tasks, as a table"},
		{"chart", map[string]any{"type": "entry", "by": "at", "period": "day", "sum": "amount", "unit": "glasses"}, "; it shows Amount of entries by At: 2 days, in glasses"},
		{"chart", map[string]any{"type": "task", "by": "done"}, "; it shows How many tasks by Done: 2 groups"},
		{"calendar", map[string]any{"type": "task"}, "; it shows 1 task by due"},
		{"tracker", map[string]any{}, "; it shows 1 habit"},
	} {
		said, isErr := call(t, a, "add_component", map[string]any{"component": c.component, "props": c.props})
		if isErr || !strings.HasSuffix(said, c.shows) {
			t.Errorf("add_component %s %v should end %q, got %q", c.component, c.props, c.shows, said)
		}
	}
}

// A look at a page lists each block on it that could only say it is set
// up wrong, and a look at a component before it is added says so too.
func TestALookListsBlocksSetUpWrong(t *testing.T) {
	a, h := newApp(t)
	id := storedBlock(t, a, "collection", setUpWrong[0].props)
	var seen struct {
		Outline struct{ Problems []string }
	}
	decode(t, get(t, h, "/api/look?path=/"), &seen)
	want := "block " + id + " (collection) cannot be shown as it is set up: " + setUpWrong[0].why
	if strings.Join(seen.Outline.Problems, "\n") != want {
		t.Errorf("the look at Home should list the broken block, and only it:\n%q\ngot %q", want, seen.Outline.Problems)
	}
	decode(t, get(t, h, "/api/look?path=/canvas/"+id), &seen)
	if !strings.Contains(strings.Join(seen.Outline.Problems, "\n"), "this block cannot be shown as it is set up: a board needs") {
		t.Errorf("the block's own page says it too, got %q", seen.Outline.Problems)
	}
	decode(t, postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"component": "calendar", "props": map[string]any{"type": "tasks"}}), &seen)
	if !strings.Contains(strings.Join(seen.Outline.Problems, "\n"), "this calendar cannot be shown as it is set up: there is no content type tasks") {
		t.Errorf("a component looked at before it is added says it could not be shown, got %q", seen.Outline.Problems)
	}
}
