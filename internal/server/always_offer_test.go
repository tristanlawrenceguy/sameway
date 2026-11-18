package server_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// The owner's own answers become a standing yes when they say so: after
// a third yes to the same kind of request from the same person, they are
// asked once whether to say it for good, and their yes puts it on that
// person's page, so the next is made without asking.
func TestRepeatedYesesOfferAStandingYes(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	sam, _ := a.Store.Create("person", map[string]any{"name": "Sam", "email": "sam@example.com", "access": "view"})
	them := a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: records.View})
	offered := func() string {
		for _, p := range a.Chat.Proposals() {
			if p.Fields["summary"] == "Always say yes to notes made by Sam?" {
				return p.ID
			}
		}
		return ""
	}
	for i, title := range []string{"Milk", "Eggs", "Bread"} {
		them.Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"`+title+`"}}`))
		for _, p := range a.Chat.Suggestions() {
			wantStatus(t, postForm(t, h, "/suggested/yes", url.Values{"id": {p.ID}}), http.StatusSeeOther)
		}
		if got := offered(); (got != "") != (i == 2) {
			t.Fatalf("after %d yeses, offered: %v", i+1, got != "")
		}
	}
	wantStatus(t, postForm(t, h, "/proposal/"+offered()+"/accept", nil), http.StatusSeeOther)
	if text, _ := them.Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"Butter"}}`)); !strings.Contains(text, "without asking") {
		t.Errorf("the next is made without asking: %s", text)
	}
}

// A standing yes is the owner's press alone: nobody asks for one, even
// with Always on changing people, and nobody asks for what Sameway keeps,
// such as their access.
func TestNobodyAsksForAStandingYes(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view", "always": []any{"update_record person"}})
	them := a.Chat.For(records.Visitor{Person: vi.ID, Name: "Vi", Access: records.View})
	for _, fields := range []string{`{"always":["update_record person","create_record event"]}`, `{"access":"edit"}`} {
		text, isErr := them.Call("update_record", json.RawMessage(`{"type":"person","id":"`+vi.ID+`","fields":`+fields+`}`))
		if !isErr || !strings.Contains(text, "not theirs to ask for") {
			t.Errorf("%s is refused, never made or put to the owner: %v %s", fields, isErr, text)
		}
	}
	if p, _ := a.Store.Get("person", vi.ID); len(anyStrings(p.Fields["always"])) != 1 || p.Fields["access"] != "view" || len(a.Chat.Suggestions()) != 0 {
		t.Errorf("nothing changed and nothing waits: %v %v %d", p.Fields["always"], p.Fields["access"], len(a.Chat.Suggestions()))
	}
	if text, _ := them.Call("update_record", json.RawMessage(`{"type":"person","id":"`+vi.ID+`","fields":{"phone":"0123"}}`)); !strings.Contains(text, "without asking") {
		t.Errorf("what Always gave them still is: %s", text)
	}
}
