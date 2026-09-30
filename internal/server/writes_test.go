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

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record is written one way from every way in: chat.WriteAs, which
// writes, keeps what was there and logs it as whoever did it. A handler
// that writes the store itself has to do all three and can forget one, as
// the API's file without content once forgot the log, so it could never
// be undone. These write the store directly for a reason, said here; a new
// one fails until it goes through WriteAs or says why it does not.
var writesTheStoreItself = map[string]string{
	"about.go nudges":               "a reminder ringing is the system's, logged as rang",
	"clock.go ring":                 "the same",
	"clock_set.go clockSet":         "an alarm or timer set, logged in its own words",
	"clock_set.go setReminder":      "a reminder dismissed or snoozed, logged in its own words",
	"track.go habitLog":             "an amount logged against a habit, logged as the habit's",
	"clash.go clashChoose":          "a version chosen after two edits at once, logged as that choice",
	"canvas.go canvasDelete":        "the canvas's own blocks, logged with the canvas's words",
	"chats.go blockPlace":           "the same",
	"props.go blockProps":           "the same",
	"collection_keep.go canvasKeep": "the same",
	"tabs.go seedChat":              "a new tab's chat block, part of making the tab",
	"keep.go keepFile":              "a file kept on disk; each way of adding one logs it as added",
	"keep.go readKept":              "a file's reading: its status and text, not anyone's change",
	"files.go readNow":              "the same",
	"files.go convertLater":         "the same",
	"files_audio.go pairCaptions":   "the same",
	"hostwrite.go readInBackground": "the same",
	"transcribe.go transcribeFile":  "the same",
	"writedown.go writePart":        "the same",
	"writedown.go finishWriting":    "the same",
	"writedown.go writeWAVHere":     "the same",
	"schema_change.go RemoveField":  "a type's shape changing, logged as the schema change",
	"schema_change.go RemoveType":   "the same",
}

func TestRecordsAreWrittenOneWay(t *testing.T) {
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
								t.Errorf("%s writes the store itself: write records through chat.WriteAs, or say here why it does not", key)
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
	a, h := newApp(t)
	trail := func(way, id, action, actor string) {
		t.Helper()
		entries, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
		for _, e := range entries {
			if e.Fields["target_id"] == id && e.Fields["action"] == action {
				if e.Fields["actor"] != actor {
					t.Errorf("%s: %s is logged as %v's, not %s's", way, action, e.Fields["actor"], actor)
				}
				if !a.Chat.Undoable(e) {
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
	trail("the API", id, "created", chat.ActorAgent)
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"title": "Changed"})
	trail("the API", id, "updated", chat.ActorAgent)
	do(t, h, http.MethodDelete, "/api/note/"+id, nil, "")
	trail("the API", id, "deleted", chat.ActorAgent)
	up := postJSON(t, h, http.MethodPost, "/api/file/upload", map[string]any{"filename": "later.pdf"})
	var stub map[string]any
	decode(t, up, &stub)
	if up.Code != http.StatusCreated {
		t.Fatalf("a file without content: %d %v", up.Code, stub)
	}
	trail("the API's file without content", stub["id"].(string), "created", chat.ActorAgent)

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
