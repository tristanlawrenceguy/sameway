package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// seedsEverywhere is "seeds" in three tasks, two notes and a block.
func seedsEverywhere(t *testing.T) (*app.App, http.Handler) {
	t.Helper()
	a, h := newApp(t)
	for _, title := range []string{"Order seeds", "Sow the seeds"} {
		a.Store.Create("note", map[string]any{"title": title})
	}
	for _, title := range []string{"Buy seeds", "Water seeds", "Label seeds"} {
		a.Store.Create("task", map[string]any{"title": title})
	}
	a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{"component": "text", "props": map[string]any{"text": "Seeds by the door"}}))
	return a, h
}

func searchCall(a *app.App, args map[string]any) (string, bool) {
	raw, _ := json.Marshal(args)
	return a.Chat.Call("search", raw)
}

// The assistant's search counts what it found by kind, narrows with type
// in the same words as the page, and refuses a kind it cannot search in.
func TestTheAssistantsSearchCountsAndNarrows(t *testing.T) {
	a, _ := seedsEverywhere(t)
	text, isErr := searchCall(a, map[string]any{"query": "seeds"})
	if isErr || !strings.HasPrefix(text, "6 found: 3 tasks, 2 notes, 1 block (") || !strings.Contains(text, "Order seeds") {
		t.Errorf("everything, counted by kind first:\n%s", text)
	}
	text, isErr = searchCall(a, map[string]any{"query": "seeds", "type": "task"})
	if isErr || !strings.HasPrefix(text, "Showing tasks only: 3 of 6 found (2 notes, 1 block elsewhere) (") || strings.Contains(text, "Order seeds") || !strings.Contains(text, "Buy seeds") {
		t.Errorf("narrowed to tasks, with what is elsewhere:\n%s", text)
	}
	for _, kind := range []string{"zebra", "block", "message"} {
		text, isErr = searchCall(a, map[string]any{"query": "seeds", "type": kind})
		if !isErr || !strings.HasPrefix(text, "There is no kind of thing called “"+kind+"” to search in; the kinds are: ") || !strings.Contains(text, "note") || !strings.Contains(text, "task") || strings.Contains(text, "Buy seeds") {
			t.Errorf("%s is refused as an error naming the kinds:\n%s", kind, text)
		}
	}
	if kinds := strings.SplitN(text, "the kinds are: ", 2)[1]; strings.Contains(kinds, "block,") || strings.Contains(kinds, "message") {
		t.Errorf("the system's own kinds are not offered: %s", kinds)
	}
	if text, _ = searchCall(a, map[string]any{"query": "zebra"}); text != `nothing has "zebra" in it` {
		t.Errorf("nothing found is said as before: %s", text)
	}
}

// Past fifty the answer says which these are and how to ask for more.
func TestTheAssistantsSearchSaysThereIsMore(t *testing.T) {
	a, _ := newApp(t)
	for i := range 55 {
		a.Store.Create("task", map[string]any{"title": fmt.Sprintf("Fern %d", i)})
	}
	one, _ := searchCall(a, map[string]any{"query": "fern"})
	if !strings.HasPrefix(one, "55 found: 55 tasks. Showing 1–50 of 55; ask for page 2 for more (") || strings.Count(one, "\ntask ") != 50 {
		t.Errorf("the first fifty, and that there are more:\n%.300s", one)
	}
	two, _ := searchCall(a, map[string]any{"query": "fern", "page": 2})
	if !strings.HasPrefix(two, "55 found: 55 tasks. Showing 51–55 of 55, the last of 2 pages (") || strings.Count(two, "\ntask ") != 5 {
		t.Errorf("page 2 is the last five:\n%s", two)
	}
}

// /api/search gives the counts and the total, narrows with type, pages,
// and refuses a kind it cannot search in with a 400.
func TestTheAPISearchCountsNarrowsAndPages(t *testing.T) {
	_, h := seedsEverywhere(t)
	var out struct {
		Count, Total, Found, Page, Pages int
		Counts                           map[string]int
		Said, Next                       string
		Hits                             []struct{ Type, Title string }
	}
	decode(t, get(t, h, "/api/search?q=seeds"), &out)
	if out.Total != 6 || out.Count != 6 || out.Counts["task"] != 3 || out.Counts["note"] != 2 || out.Counts["block"] != 1 || out.Said != "6 found: 3 tasks, 2 notes, 1 block" || out.Next != "" {
		t.Errorf("everything, counted: %+v", out)
	}
	out.Hits = nil
	decode(t, get(t, h, "/api/search?q=seeds&type=task"), &out)
	if out.Total != 6 || out.Found != 3 || len(out.Hits) != 3 || out.Hits[0].Type != "task" || out.Counts["note"] != 2 {
		t.Errorf("tasks only, with the counts of everything: %+v", out)
	}
	bad := get(t, h, "/api/search?q=seeds&type=zebra")
	wantStatus(t, bad, http.StatusBadRequest)
	if body := bad.Body.String(); !strings.Contains(body, "There is no kind of thing called “zebra” to search in; the kinds are: ") || strings.Contains(body, "Buy seeds") {
		t.Errorf("a kind there is none of is refused: %s", body)
	}

	_, h = newApp(t)
	for i := range 55 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": fmt.Sprintf("Fern %d", i)}), http.StatusCreated)
	}
	out.Next = ""
	decode(t, get(t, h, "/api/search?q=fern"), &out)
	if out.Count != 50 || out.Found != 55 || out.Pages != 2 || out.Next != "/api/search?page=2&q=fern" {
		t.Errorf("fifty, and the way to the rest: %+v", out)
	}
	next := out.Next
	out.Next = ""
	decode(t, get(t, h, next), &out)
	if out.Count != 5 || out.Page != 2 || out.Next != "" {
		t.Errorf("the last five: %+v", out)
	}
}

// The internet's search counts only what is published, narrows only to a
// published kind, and keeps results as ChatGPT's connectors read them.
func TestPublishedSearchCountsAndNarrows(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	n, _ := a.Store.Create("note", map[string]any{"title": "Sourdough", "body": "Flour, water and salt."})
	a.Store.Create("task", map[string]any{"title": "Sourdough errand"})
	a.Workspace.Config.Publish.Types = "note"
	pub := srv.Public(&mcp.Server{App: a, Version: "test", Published: func() map[string]bool { return srv.Published().Types }})
	call := func(args string) (res struct {
		Result struct {
			IsError    bool
			Content    []struct{ Text string }
			Structured struct {
				Results []struct{ ID, Title, URL string }
				Counts  map[string]int
				Total   int
				Said    string
			} `json:"structuredContent"`
		}
	}) {
		body := public(t, pub, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":`+args+`}}`).Body.Bytes()
		json.Unmarshal(body, &res)
		return res
	}
	all := call(`{"query":"sourdough"}`).Result
	if all.IsError || all.Structured.Total != 1 || all.Structured.Counts["note"] != 1 || all.Structured.Counts["task"] != 0 || all.Structured.Said != "1 found: 1 note" ||
		len(all.Structured.Results) != 1 || all.Structured.Results[0].ID != "note/"+n.ID || !strings.HasSuffix(all.Structured.Results[0].URL, "/t/note/"+n.ID) {
		t.Errorf("the published note, counted, and nothing unpublished: %+v", all)
	}
	if notes := call(`{"query":"sourdough","type":"note"}`).Result; notes.IsError || len(notes.Structured.Results) != 1 {
		t.Errorf("narrowed to a published kind: %+v", notes)
	}
	for _, kind := range []string{"task", "zebra"} {
		r := call(`{"query":"sourdough","type":"` + kind + `"}`).Result
		if !r.IsError || len(r.Content) != 1 || r.Content[0].Text != "There is no kind of thing called “"+kind+"” to search in; the kinds are: note. Leave out type to search everything." {
			t.Errorf("%s is not published, so it is refused: %+v", kind, r)
		}
	}
}
