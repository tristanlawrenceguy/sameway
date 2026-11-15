package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// The assistant working for someone the owner let in has their
// permissions, never the owner's: what is the owner's alone is out of its
// reach by every tool, as it is out of reach of their pages and their
// search, and for someone who may only look it is given only what reads.
func TestTheAssistantHasTheAskersPermissions(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	bob, _ := a.Store.Create("person", map[string]any{"name": "Bob", "email": "bob@example.com", "access": "edit"})
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view"})
	kept, _ := a.Store.Create(records.EnquiryType, map[string]any{"question": "When you are free for the surprise party", "who": bob.ID, "state": "answered"})
	a.Store.Create("note", map[string]any{"title": "Surprise party shopping"})
	editor := records.Visitor{Person: bob.ID, Name: "Bob", Login: "bob@example.com", Access: records.Edit}
	viewer := records.Visitor{Person: vi.ID, Name: "Vi", Login: "vi@example.com", Access: records.View}

	ed := a.Chat.For(editor)
	for _, c := range [][2]string{{"find_records", `{"type":"enquiry"}`}, {"get_record", `{"type":"enquiry","id":"` + kept.ID + `"}`}} {
		if text, isErr := ed.Call(c[0], json.RawMessage(c[1])); !isErr || strings.Contains(text, "surprise") {
			t.Errorf("%s %s is the owner's alone: %v %s", c[0], c[1], isErr, text)
		}
	}
	if text, _ := ed.Call("search", json.RawMessage(`{"query":"surprise party"}`)); strings.Contains(text, "enquiry") || !strings.Contains(text, "Surprise party shopping") {
		t.Errorf("the editor's search finds what they may read and not the owner's: %s", text)
	}
	if text, _ := ed.Call("get_record", json.RawMessage(`{"type":"person","id":"`+bob.ID+`"}`)); strings.Contains(text, `"type": "enquiry"`) {
		t.Errorf("what a record connects to that is the owner's is not shown: %s", text)
	}
	if text, _ := a.Chat.Call("search", json.RawMessage(`{"query":"surprise party"}`)); !strings.Contains(text, "enquiry") {
		t.Errorf("the owner's assistant still reaches everything: %s", text)
	}

	look := a.Chat.For(viewer)
	for _, tl := range look.Tools() {
		if tl.Name == "set_setting" {
			t.Errorf("someone who may only look is given the owner's %s", tl.Name)
		}
	}
	if text, _ := look.Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"From Vi"}}`)); !strings.Contains(text, "Nothing has changed") {
		t.Errorf("a change they ask for is not made: %s", text)
	}
	if _, isErr := look.Call("create_record", json.RawMessage(`{"type":"enquiry","fields":{"question":"x"}}`)); !isErr {
		t.Error("nor asked for, of a kind they cannot reach")
	}
	if _, isErr := look.Call("find_records", json.RawMessage(`{"type":"note"}`)); isErr {
		t.Error("they may still read")
	}

	page := as(t, h, editor, http.MethodGet, "/search?q=surprise", "", "").Body.String()
	if strings.Contains(page, "/t/enquiry/") || !strings.Contains(page, "shopping") {
		t.Errorf("an editor's search page finds what they may read and not the owner's: enquiry %v, note %v", strings.Contains(page, "/t/enquiry/"), strings.Contains(page, "shopping"))
	}
}
