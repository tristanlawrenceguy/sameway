package server_test

import (
	"net/http"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Every route says who may use it; one nobody said anything about is the
// owner's alone until somebody does, rather than open to everyone let in.
// The mux is built from the table alone (routes.go), and a request whose
// route is not in it is the owner's (ownerOnlyRequest).
func TestEveryRouteSaysWhoMayUseIt(t *testing.T) {
	t.Parallel()
	for _, r := range server.Routes() {
		if r.Access == "" {
			t.Errorf("%s: who may use it? Say so in its route (routes*.go): people, or owner", r.Pattern)
		}
	}
}

// Someone let in to look reads what is shared and nothing of the owner's
// own: not the log or the chats, however they are asked for. An export of
// them is refused here too, not only by the export's own rule that
// internal kinds are not taken out.
func TestAVisitorCannotTakeTheOwnersRecordsAway(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	viewer := records.Visitor{Name: "Vi", Login: "vi@example.com", Access: records.View}
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
	t.Parallel()
	a, h := newApp(t)
	editor := records.Visitor{Name: "Bob", Login: "bob@example.com", Access: records.Edit}
	viewer := records.Visitor{Name: "Vi", Login: "vi@example.com", Access: records.View}
	body := `{"name":"plant","description":"A plant in the garden","title":"name","fields":[{"name":"name","type":"string"}]}`
	if res := as(t, h, viewer, http.MethodPost, "/api/types", body, "application/json"); res.Code != http.StatusForbidden {
		t.Errorf("a viewer adds a type over the API: %d", res.Code)
	}
	if res := as(t, h, editor, http.MethodPost, "/api/types", body, "application/json"); res.Code >= 300 {
		t.Errorf("an editor adds a type over the API: %d %s", res.Code, res.Body.String())
	}
	has := func(v records.Visitor, name string) bool {
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
