package server_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record is written one way from every way in: as a change's ops, by
// records.WriteAs or Apply (s.apply on a page), which write, keep what was
// there and log it as whoever did it. A handler that writes the store
// itself has to do all three and can forget one, as the API's file without
// content once forgot the log, so it could never be undone. The system's
// own bookkeeping (a file's reading, a reminder ringing, a new tab's chat,
// the example's blocks) goes through records.ApplyOps too, unlogged. What
// still writes the store directly does so for a reason, said here; a new
// one fails until it goes through Apply or says why it does not.
var writesTheStoreItself = map[string]string{
	"schema_change.go RemoveType": "a type deleted drops its whole table in one statement as the type goes, logged as the schema change; its records go with their type, not one by one",
}

func TestRecordsAreWrittenOneWay(t *testing.T) {
	t.Parallel()
	found := map[string]bool{}
	for _, dir := range []string{".", "../cli", "../mcp", "../app"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), f, src, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range file.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if x, ok := sel.X.(*ast.SelectorExpr); ok && x.Sel.Name == "Store" {
						switch sel.Sel.Name {
						case "Create", "Update", "Delete", "DeleteAll":
							key := filepath.Base(f) + " " + fn.Name.Name
							found[key] = true
							if writesTheStoreItself[key] == "" {
								t.Errorf("%s writes the store itself: write records through records.WriteAs or Apply, or say here why it does not", key)
							}
						}
					}
					return true
				})
			}
		}
	}
	for key := range writesTheStoreItself {
		if !found[key] {
			t.Errorf("%s no longer writes the store itself; take it off the list", key)
		}
	}
}

// Made, changed and deleted from a page, over the API and by the
// assistant, a record leaves the same trail each time: an entry in the
// log, by whoever did it, that can be undone.
func TestEveryWayInLeavesTheSameTrail(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	trail := func(way, id, action, actor string) {
		t.Helper()
		entries, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
		for _, e := range entries {
			if e.Fields["target_id"] == id && e.Fields["action"] == action {
				if e.Fields["actor"] != actor {
					t.Errorf("%s: %s is logged as %v's, not %s's", way, action, e.Fields["actor"], actor)
				}
				if !a.Records.Undoable(e) {
					t.Errorf("%s: %s cannot be undone", way, action)
				}
				return
			}
		}
		t.Errorf("%s: %s left nothing in the log", way, action)
	}

	// A page.
	res := postForm(t, h, "/t/note/add", url.Values{})
	id := strings.TrimPrefix(strings.SplitN(res.Header().Get("Location"), "?", 2)[0], "/t/note/")
	trail("a page", id, "created", "human")
	postForm(t, h, "/t/note/"+id+"/props", url.Values{"prop-title": {"From a page"}})
	trail("a page", id, "updated", "human")
	postForm(t, h, "/t/note/"+id+"/delete", url.Values{})
	trail("a page", id, "deleted", "human")

	// The API.
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "From the API"}), &made)
	id, _ = made["id"].(string)
	trail("the API", id, "created", records.ActorAgent)
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"title": "Changed"})
	trail("the API", id, "updated", records.ActorAgent)
	do(t, h, http.MethodDelete, "/api/note/"+id, nil, "")
	trail("the API", id, "deleted", records.ActorAgent)
	up := postJSON(t, h, http.MethodPost, "/api/file/upload", map[string]any{"filename": "later.pdf"})
	var stub map[string]any
	decode(t, up, &stub)
	if up.Code != http.StatusCreated {
		t.Fatalf("a file without content: %d %v", up.Code, stub)
	}
	trail("the API's file without content", stub["id"].(string), "created", records.ActorAgent)

	// The assistant.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "From the assistant"}}),
		{Text: "Made."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"a note"}, "from": {"/chat"}})
	notes, _ := a.Store.List("note", store.ListOptions{})
	for _, n := range notes {
		if n.Fields["title"] == "From the assistant" {
			trail("the assistant", n.ID, "created", "assistant")
		}
	}
}
