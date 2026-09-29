package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Every route says who may use it; one nobody said anything about is the
// owner's alone until somebody does, rather than open to everyone let in.
func TestEveryRouteSaysWhoMayUseIt(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	pattern := regexp.MustCompile(`(?:HandleFunc|Handle)\("([^"]*)"`)
	routes := map[string]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, _ := os.ReadFile(f)
		for _, m := range pattern.FindAllStringSubmatch(string(src), -1) {
			routes[m[1]] = true
		}
	}
	said := server.RouteAccess()
	for route := range routes {
		if _, ok := said[route]; !ok {
			t.Errorf("%s: who may use it? Say so in routeAccess (access_routes.go): people, or owner", route)
		}
	}
	for route := range said {
		if !routes[route] {
			t.Errorf("%s is no longer a route; take it out of routeAccess", route)
		}
	}
}

// Someone let in to look reads what is shared and nothing of the owner's
// own: not the log or the chats, however they are asked for. An export of
// them is refused here too, not only by the export's own rule that
// internal kinds are not taken out.
func TestAVisitorCannotTakeTheOwnersRecordsAway(t *testing.T) {
	_, h := newApp(t)
	viewer := chat.Visitor{Name: "Vi", Login: "vi@example.com", Access: chat.View}
	for _, path := range []string{"/export/activity.csv", "/export/message.xlsx", "/export/conversation.csv", "/t/activity", "/api/activity", "/api/workspaces", "/workspaces"} {
		if res := as(t, h, viewer, http.MethodGet, path, "", ""); res.Code != http.StatusForbidden {
			t.Errorf("a viewer reads %s: %d", path, res.Code)
		}
	}
	if res := as(t, h, viewer, http.MethodGet, "/export/note.csv", "", ""); res.Code != http.StatusOK {
		t.Errorf("a viewer keeps the notes they can read: %d", res.Code)
	}
}

// Changing the workspace's shape is for everyone who may change it, the
// same over the API as through the assistant and MCP, which offer an
// editor add_type and add_field and a viewer neither.
func TestTheShapeIsAnEditorsOverEveryWayIn(t *testing.T) {
	a, h := newApp(t)
	editor := chat.Visitor{Name: "Bob", Login: "bob@example.com", Access: chat.Edit}
	viewer := chat.Visitor{Name: "Vi", Login: "vi@example.com", Access: chat.View}
	body := `{"name":"plant","description":"A plant in the garden","title":"name","fields":[{"name":"name","type":"string"}]}`
	if res := as(t, h, viewer, http.MethodPost, "/api/types", body, "application/json"); res.Code != http.StatusForbidden {
		t.Errorf("a viewer adds a type over the API: %d", res.Code)
	}
	if res := as(t, h, editor, http.MethodPost, "/api/types", body, "application/json"); res.Code >= 300 {
		t.Errorf("an editor adds a type over the API: %d %s", res.Code, res.Body.String())
	}
	has := func(v chat.Visitor, name string) bool {
		for _, tool := range a.Chat.For(v).Tools() {
			if tool.Name == name {
				return true
			}
		}
		return false
	}
	if !has(editor, "add_type") || !has(editor, "add_field") {
		t.Error("an editor's assistant, and so their MCP connection, can change the shape too")
	}
}
