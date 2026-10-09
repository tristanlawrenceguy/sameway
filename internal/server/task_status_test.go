package server_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// "A board of my project's tasks by status" is the example the prompt,
// the tools and the manifests give, so a task has a status: To do, Doing,
// Done, one fact with its done tick. Both models of an evaluation failed
// the request when it had only the tick.

// column is how a board heads a column holding n cards.
func column(label, n string) string {
	return label + ` <span class="sw-collection__count">(` + n + `)</span>`
}

func TestTasksMakeABoardByStatus(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	ids := map[string]string{}
	for _, task := range []map[string]any{
		{"title": "Order compost"},
		{"title": "Dig the pond", "status": "doing"},
		{"title": "Sow beans", "done": true},
		{"title": "Plant garlic", "status": "done"},
	} {
		rec, err := a.Store.Create("task", task)
		if err != nil {
			t.Fatal(err)
		}
		ids[task["title"].(string)] = rec.ID
	}
	sow, _ := a.Store.Get("task", ids["Sow beans"])
	garlic, _ := a.Store.Get("task", ids["Plant garlic"])
	if sow.Fields["status"] != "done" || garlic.Fields["done"] != true {
		t.Errorf("given one, a new task gets the other: ticked is Done, Done is ticked; got %v and %v", sow.Fields["status"], garlic.Fields["done"])
	}

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "as": "board", "by": "status", "label": "Tasks"},
	}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	todo, doing, done := strings.Index(page, column("To do", "1")), strings.Index(page, column("Doing", "1")), strings.Index(page, column("Done", "2"))
	if todo < 0 || doing < todo || done < doing {
		t.Fatalf("the board has To do (1), Doing (1), Done (2): %s", said(page))
	}
	if !strings.Contains(page, `name="prop-status"><option value="todo" selected>To do</option><option value="doing">Doing</option><option value="done">Done</option></select>`) {
		t.Error("each card's Move offers the choices by their labels")
	}
	if strings.Contains(page, "has no pick-list") {
		t.Error("a task has a pick-list for a board")
	}

	// Moved to Done on the board, it is ticked.
	moved := after(t, h, doWithReferer(t, h, "/t/task/"+ids["Order compost"]+"/props", url.Values{"prop-status": {"done"}}, "http://example.com/")).Body.String()
	if !strings.Contains(moved, "Order compost moved from To do to Done.") {
		t.Errorf("the move says where it went: %s", said(moved))
	}
	if rec, _ := a.Store.Get("task", ids["Order compost"]); rec.Fields["done"] != true {
		t.Error("a task moved to Done is ticked done")
	}

	// Unticked, a Done task goes back to To do.
	postForm(t, h, "/t/task/"+ids["Sow beans"]+"/props", url.Values{"prop-done": {"false"}})
	if rec, _ := a.Store.Get("task", ids["Sow beans"]); rec.Fields["status"] != "todo" || rec.Fields["done"] != false {
		t.Errorf("unticked, a Done task is To do, got %v", rec.Fields)
	}

	// Ticked from Doing it is Done; Undo puts back Doing, not To do.
	postForm(t, h, "/t/task/"+ids["Dig the pond"]+"/props", url.Values{"prop-done": {"true"}})
	if rec, _ := a.Store.Get("task", ids["Dig the pond"]); rec.Fields["status"] != "done" {
		t.Errorf("ticked, a task is Done, got %v", rec.Fields["status"])
	}
	if err := a.Records.UndoAs("owner", ""); err != nil {
		t.Fatal(err)
	}
	if rec, _ := a.Store.Get("task", ids["Dig the pond"]); rec.Fields["status"] != "doing" || rec.Fields["done"] != false {
		t.Errorf("undoing the tick puts back Doing, got %v", rec.Fields)
	}

	// A list of tasks is narrowed by status, by its labels, and the API
	// sees the same.
	id := addCollection(t, h, map[string]any{"type": "task", "label": "All tasks", "controls": true})
	list := section(t, get(t, h, "/").Body.String(), id)
	for _, want := range []string{`name="c-` + id + `-status"`, `>To do<`, `>Doing<`, `>Done<`} {
		if !strings.Contains(list, want) {
			t.Errorf("the list offers status by its labels: missing %s in %.2000s", want, list)
		}
	}
	var out struct{ Count int }
	decode(t, get(t, h, "/api/task?where=status%3Ddone"), &out)
	if out.Count != 2 {
		t.Errorf("Order compost and Plant garlic are Done, got %d", out.Count)
	}
}

// A repeating task moved to Done, like one ticked, is due again at once,
// back in To do, and says so.
func TestARepeatingTaskMovedToDoneIsToDoAgain(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	fern, err := a.Store.Create("task", map[string]any{"title": "Water the fern", "due": "2026-10-06T00:00:00Z", "repeat": "every Tuesday"})
	if err != nil {
		t.Fatal(err)
	}
	moved := after(t, h, doWithReferer(t, h, "/t/task/"+fern.ID+"/props", url.Values{"prop-status": {"done"}}, "http://example.com/")).Body.String()
	rec, _ := a.Store.Get("task", fern.ID)
	if rec.Fields["status"] != "todo" || rec.Fields["done"] != false || rec.Fields["due"] == "2026-10-06T00:00:00Z" {
		t.Errorf("moved to Done, it is due again and To do: %v", rec.Fields)
	}
	if !strings.Contains(moved, "Water the fern is done.") || !strings.Contains(moved, "It repeats every Tuesday") {
		t.Errorf("it says it is done and due again: %s", said(moved))
	}
	postForm(t, h, "/t/task/"+fern.ID+"/props", url.Values{"prop-done": {"true"}})
	if rec, _ := a.Store.Get("task", fern.ID); rec.Fields["status"] != "todo" || rec.Fields["done"] != false {
		t.Errorf("ticked, it is due again and To do: %v", rec.Fields)
	}
}

// A workspace made before tasks had a status gets the field, and each
// task it already had reads its status from its tick, the same on the
// board, in a filter and through the API, with nothing rewritten.
func TestAnOlderWorkspaceBoardsItsTasksByStatus(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	known := filepath.Join(t.TempDir(), "known.json")
	os.WriteFile(known, []byte("[]"), 0o644)
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	old := "name: task\ntitle: title\nprovided: true\nfields:\n  title:\n    type: string\n  done:\n    type: bool\n  due:\n    type: datetime\n"
	if err := os.WriteFile(filepath.Join(dir, "schema", "task.yaml"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	then, err := schema.Load(ws.SchemaDir())
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ws.DBPath(), then)
	if err != nil {
		t.Fatal(err)
	}
	st.Create("task", map[string]any{"title": "Sow beans", "done": true})
	open, _ := st.Create("task", map[string]any{"title": "Order compost"})
	st.Close()

	a, err := app.Open(dir, app.Options{Machine: workspace.Machine{Known: known}})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.Chat.Provider, a.Chat.ProviderErr = nil, llm.ErrNotConfigured
	h := server.New(a)

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "as": "board", "by": "status", "label": "Tasks"},
	}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, column("To do", "1")) || !strings.Contains(page, column("Done", "1")) {
		t.Errorf("the older tasks are on the board by their ticks: %s", said(page))
	}
	var out struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/task?where=status%3Ddone"), &out)
	if len(out.Records) != 1 || out.Records[0].Fields["title"] != "Sow beans" {
		t.Errorf("the API reads the same status: %+v", out.Records)
	}
	postForm(t, h, "/t/task/"+open.ID+"/props", url.Values{"prop-done": {"true"}})
	if rec, _ := a.Store.Get("task", open.ID); rec.Fields["status"] != "done" {
		t.Errorf("ticked, an older task is Done, got %v", rec.Fields["status"])
	}
}

// Adding a field says what the records already there got, over the API
// and from the assistant's tool.
func TestAddingAFieldSaysWhatExistingRecordsGet(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	for _, title := range []string{"Order compost", "Dig the pond", "Sow beans"} {
		a.Store.Create("task", map[string]any{"title": title})
	}
	var added struct{ Existing string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/types/task/fields", map[string]any{"name": "priority", "type": "enum", "values": []string{"low", "high"}, "default": "low"}), &added)
	if added.Existing != "3 existing tasks get priority Low" {
		t.Errorf("the API says what the tasks got: %q", added.Existing)
	}

	model := &scripted{steps: []*llm.Response{
		toolCall("add_field", map[string]any{"type": "task", "name": "effort", "kind": "int"}),
		{Text: "Tasks have an effort now."},
	}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"give tasks an effort"}}), http.StatusSeeOther)
	if len(model.seen) < 2 {
		t.Fatalf("the model was asked %d times", len(model.seen))
	}
	if got := model.seen[1].Messages[len(model.seen[1].Messages)-1].ToolResults[0].Content; !strings.Contains(got, "(3 existing tasks get effort 0)") {
		t.Errorf("the tool says what the tasks got: %q", got)
	}
}
