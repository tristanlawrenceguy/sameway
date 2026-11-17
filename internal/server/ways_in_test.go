package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// How people reach the owner is one question: each way in and whether it
// is open, the standing yes, and the requests of the last month by way
// in, answered and how soon, and what waits.
func TestHowPeopleReachYouIsOneQuestion(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	a.Workspace.Config.Publish.Types = "note"
	a.Workspace.Config.Publish.Book = "on"
	form := url.Values{"at": {thursdayAt(9).UTC().Format(time.RFC3339)}, "name": {"Pat"}, "email": {"pat@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/book", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	internet.ServeHTTP(httptest.NewRecorder(), req)
	wantStatus(t, postForm(t, h, "/suggested/yes", url.Values{"id": {a.Chat.Suggestions()[0].ID}}), http.StatusSeeOther)
	sam, _ := a.Store.Create("person", map[string]any{"name": "Sam", "access": "view"})
	a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: records.View}).Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"Milk"}}`))

	text, isErr := a.Chat.Call("ways_in", json.RawMessage(`{}`))
	for _, want := range []string{"Email: not connected", "Published pages: notes", "Booking link: on", "Telegram: not connected", "Other agents: find you", "Standing yes: none",
		"Through your booking link: 1 requests in 30 days, 1 answered, usually within", "Asked through the assistant: 1 requests in 30 days, 0 answered, 1 waiting"} {
		if isErr || !strings.Contains(text, want) {
			t.Errorf("it says %q:\n%s", want, text)
		}
	}
	if text, isErr := a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: records.View}).Call("ways_in", json.RawMessage(`{}`)); !isErr {
		t.Errorf("it is the owner's to ask: %s", text)
	}
}
