package server_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What a person does from a page, the assistant can do with a tool, or it
// is theirs alone by design. Each page action says which: the tools that
// do the same, or "a person's: ..." and why. There is no third answer: a
// page action with neither is a gap, and a new one fails here until the
// assistant has a tool for it or the reason it should not is said.
var pageActionTools = map[string]string{
	"/t/{type}/add":               "create_record",
	"/t/{type}/{id}/props":        "update_record",
	"/t/{type}/{id}/discard":      "undo_change",
	"/t/{type}/{id}/delete":       "a person's: a record goes when a person deletes it, or when its making is undone",
	"/t/{type}/import":            "import_records",
	"/t/{type}/import/{file}/run": "import_records",
	"/t/file/upload":              "a person's: a file comes from their computer; the assistant reads files already added",
	"/files/{id}/transcribe":      "write_down",
	"/activity/{id}/undo":         "undo_change",
	"/act/{id}":                   "run_action",
	"/canvas/{id}/place":          "update_component",
	"/canvas/{id}/props":          "update_component",
	"/canvas/{id}/delete":         "remove_component",
	"/help/set":                   "set_setting",
	"/model/use":                  "set_setting",
	"/model/check":                "a person's: checking the model is checking the assistant itself",
	"/clock/set":                  "create_record",
	"/clock/{id}/done":            "update_record",
	"/clock/{id}/snooze":          "update_record",
	"/habit/{id}/log":             "create_record",
	"/chat":                       "a person's: it is what they say to the assistant",
	"/chat/stream":                "a person's: the same",
	"/chat/stop":                  "a person's: stopping the assistant",
	"/chat/new":                   "a person's: which conversation they are in is theirs",
	"/chat/open":                  "a person's: the same",
	"/chat/delete":                "a person's: the same",
	"/chat/clear":                 "clear_conversation",
	"/proposal/{id}/accept":       "a person's: the answer to the assistant's own question",
	"/proposal/{id}/dismiss":      "a person's: the same",
	"/proposal/{id}/instead":      "a person's: the same",
	"/clash/{id}/use":             "a person's: which of two versions to keep",
	"/clash/{id}/both":            "a person's: the same",
	"/clash/{id}/keep":            "a person's: the same",
	"/since/seen":                 "a person's: what they have seen",
	"/speech/get":                 "a person's: a download they are asked about",
	"/dictate":                    "a person's: their voice",
	"/sync":                       "a person's: computers exchanging changes, not a change",
	"/workspaces/new":             "add_workspace",
	"/workspaces/copy":            "add_workspace",
	"/workspaces/start":           "open_workspace",
	"/workspaces/restore":         "restore_workspace",
	"/workspaces/delete":          "a person's: deleting a whole workspace is its owner's",
}

func TestEveryPageActionIsTheAssistantsOrSaysWhyNot(t *testing.T) {
	a, _ := newApp(t)
	tools := map[string]bool{}
	for _, tool := range a.Describe().Tools {
		tools[tool.Name] = true
	}
	files, _ := filepath.Glob("*.go")
	pattern := regexp.MustCompile(`HandleFunc\("POST (/[^"]*)"`)
	seen := map[string]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, _ := os.ReadFile(f)
		for _, m := range pattern.FindAllStringSubmatch(string(src), -1) {
			route := m[1]
			if strings.HasPrefix(route, "/api/") || route == "/mcp" || strings.HasPrefix(route, "/hook/") {
				continue
			}
			seen[route] = true
			said, ok := pageActionTools[route]
			switch {
			case !ok:
				t.Errorf("POST %s: which tool does the same for the assistant, or why is it a person's? Say so in pageActionTools", route)
			case strings.HasPrefix(said, "a person's: "):
			default:
				for _, name := range strings.Split(said, ",") {
					if !tools[strings.TrimSpace(name)] {
						t.Errorf("POST %s names the tool %q, which the assistant does not have", route, name)
					}
				}
			}
		}
	}
	var gone []string
	for route := range pageActionTools {
		if !seen[route] {
			gone = append(gone, route)
		}
	}
	sort.Strings(gone)
	for _, route := range gone {
		t.Errorf("POST %s is no longer a page action; take it off pageActionTools", route)
	}
}

// The assistant does no more than the pages let the one it speaks for do:
// a page action that is the owner's alone has only owner's tools, so
// someone let in cannot ask their assistant for what their pages refuse
// them. The other way round, a person may do on a page what their
// assistant may not, for a reason said here.
var narrowerForTheAssistant = map[string]string{
	"/t/{type}/{id}/discard": "undo_change takes back anyone's change, so it is the owner's; a person discards only the record they just added",
}

func TestTheAssistantIsNoWiderThanThePages(t *testing.T) {
	access := server.RouteAccess()
	for route, said := range pageActionTools {
		if strings.HasPrefix(said, "a person's: ") {
			continue
		}
		owners := access["POST "+route] == "owner"
		for _, tool := range strings.Split(said, ",") {
			tool = strings.TrimSpace(tool)
			alone := chat.OwnersAlone(tool)
			switch {
			case owners && !alone:
				t.Errorf("POST %s is the owner's, but %s is anyone's: someone let in could ask for what the page refuses them", route, tool)
			case !owners && alone && narrowerForTheAssistant[route] == "":
				t.Errorf("POST %s is anyone's, but %s is the owner's: say why in narrowerForTheAssistant, or open the tool", route, tool)
			}
		}
	}
}
