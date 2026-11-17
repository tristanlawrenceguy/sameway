package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Whoever books through the published page hears back by email, from the
// owner's own mailbox, without Sameway of their own: the owner answering
// a request is something an action waits for, the assistant it asks
// writes to them with send_email, and every email waits for the owner's
// yes, word for word, before it goes.
func TestWhoeverBooksHearsBackByEmail(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	a.Workspace.Config.Publish.Book = "on"
	internet := h.(*server.Server).Public(nil)

	if text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"Hi","text":"Hello"}`)); !isErr || !strings.Contains(text, "Email in") {
		t.Fatalf("with no mailbox, the assistant is told where the person connects one: %v %s", isErr, text)
	}
	sent := make(chan string, 2)
	a.Chat.MailFrom = func() string { return "tris@example.com" }
	a.Chat.Mail = func(m chat.Outgoing) (string, error) {
		sent <- m.To + "|" + m.Subject + "|" + m.Text
		return "<1@example.com>", nil
	}

	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("send_email", map[string]any{"to": "Pat Lee <pat@example.com>", "subject": "Your visit on Thursday", "text": "Yes, see you Thursday at 9am."}),
		{Text: "I have written to Pat; it waits for your yes."},
	}}
	a.Store.Create("action", map[string]any{"title": "Write back to whoever asks", "kind": "message", "message": "I said yes to {{asked}}, asked by {{asked_by}}. Write to them.", "when": "changed", "what": "proposal", "only": []any{"by_action=request", "state=accepted"}})
	a.Automate.StartWorker()

	form := url.Values{"at": {thursdayAt(9).UTC().Format(time.RFC3339)}, "name": {"Pat Lee"}, "email": {"pat@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/book", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	internet.ServeHTTP(httptest.NewRecorder(), req)
	asks := a.Chat.Suggestions()
	if len(asks) != 1 {
		t.Fatalf("the booking waits: %d", len(asks))
	}
	wantStatus(t, postForm(t, h, "/suggested/yes", url.Values{"id": {asks[0].ID}}), http.StatusSeeOther)

	var email string
	for deadline := time.Now().Add(5 * time.Second); email == ""; {
		for _, p := range a.Chat.Proposals() {
			if s, _ := p.Fields["summary"].(string); strings.HasPrefix(s, "Email Pat Lee <pat@example.com> this?") {
				email = p.ID
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("answering the request set off the action, and the email it wrote is put to the owner")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(sent) != 0 {
		t.Fatal("nothing is sent before the owner's yes")
	}
	if today := get(t, h, "/today").Body.String(); !strings.Contains(today, "Yes, see you Thursday at 9am.") || !strings.Contains(today, "Send it") {
		t.Errorf("the owner reads it word for word: %s", truncate(today))
	}
	wantStatus(t, postForm(t, h, "/proposal/"+email+"/accept", nil), http.StatusSeeOther)
	if got := <-sent; got != "Pat Lee <pat@example.com>|Your visit on Thursday|Yes, see you Thursday at 9am." {
		t.Errorf("with the yes it goes: %s", got)
	}
}

// A message carries choices whoever gets it can take: the owner cannot
// do the hour Pat asked for, the assistant pencils in another and writes
// to Pat with a link to confirm it and one to say it does not suit. Each
// is a change the owner agreed to with the email; Pat's press makes it,
// once, the other closes, and the owner is told.
func TestAMessageCarriesChoicesTheyCanTake(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	srv := h.(*server.Server)
	internet := srv.Public(nil)
	sent := make(chan string, 1)
	a.Chat.MailFrom = func() string { return "tris@example.com" }
	a.Chat.Mail = func(m chat.Outgoing) (string, error) { sent <- m.Text; return "<1@example.com>", nil }
	rang := make(chan string, 4)
	srv.OnRing(func(title, text, _ string) { rang <- title + ": " + text })

	tentative, _ := a.Store.Create("event", map[string]any{"title": "Meeting with Pat Lee (tentative)", "starts": "2026-03-13T14:00:00Z"})
	a.Chat.Address = func() string { return "" } // nothing published
	if text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"x","text":"y","choices":[{"label":"Go","tool":"update_record","type":"event","id":"`+tentative.ID+`","fields":{"title":"z"}}]}`)); !isErr || !strings.Contains(text, "publish something") {
		t.Errorf("with nothing published, no one could press them, and it says so: %v %s", isErr, text)
	}
	a.Chat.Address = func() string { return "https://tris.example.ts.net/" }
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("send_email", map[string]any{"to": "Pat Lee <pat@example.com>", "subject": "Thursday does not work", "text": "Thursday is taken, sorry. Could you do Friday at 2pm?",
			"choices": []any{
				map[string]any{"label": "Friday at 2pm is fine", "tool": "update_record", "type": "event", "id": tentative.ID, "fields": map[string]any{"title": "Meeting with Pat Lee"}},
				map[string]any{"label": "Friday does not suit", "tool": "update_record", "type": "event", "id": tentative.ID, "fields": map[string]any{"title": "Pat cannot do Friday"}},
			}}),
		{Text: "I have written to Pat, with links to confirm or not; it waits for your yes."},
	}}
	if _, err := a.Chat.Send(context.Background(), "I can't do Thursday 9 for Pat, offer Friday at 2pm"); err != nil {
		t.Fatal(err)
	}
	ps := a.Chat.Proposals()
	if len(ps) != 1 || !strings.Contains(ps[0].Fields["detail"].(string), `With a link, Friday at 2pm is fine: pressing it would change the event "Meeting with Pat Lee (tentative)": "Meeting with Pat Lee".`) {
		t.Fatalf("the owner reads what each choice would do: %q", ps[0].Fields["detail"])
	}
	wantStatus(t, postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	text := <-sent
	var yes, no string
	for _, line := range strings.Split(text, "\n") {
		if l, ok := strings.CutPrefix(line, "Friday at 2pm is fine: https://tris.example.ts.net"); ok {
			yes = l
		}
		if l, ok := strings.CutPrefix(line, "Friday does not suit: https://tris.example.ts.net"); ok {
			no = l
		}
	}
	if yes == "" || no == "" || !strings.HasPrefix(text, "Thursday is taken, sorry.") {
		t.Fatalf("the email carries a link for each choice: %s", text)
	}

	page := public(t, internet, http.MethodGet, yes, "").Body.String()
	if !strings.Contains(page, "Friday at 2pm is fine") || !strings.Contains(page, `action="`+yes+`"`) {
		t.Fatalf("its page says what it does, with one button: %s", truncate(page))
	}
	if e, _ := a.Store.Get("event", tentative.ID); e.Fields["title"] != "Meeting with Pat Lee (tentative)" {
		t.Fatal("opening the link changes nothing")
	}
	if r := public(t, internet, http.MethodPost, yes, ""); r.Code >= 400 {
		t.Fatalf("pressing it takes it: %d %s", r.Code, r.Body.String())
	}
	if e, _ := a.Store.Get("event", tentative.ID); e.Fields["title"] != "Meeting with Pat Lee" {
		t.Errorf("as the owner agreed: %v", e.Fields["title"])
	}
	if got := <-rang; !strings.Contains(got, "Pat Lee <pat@example.com> chose Friday at 2pm is fine") {
		t.Errorf("the owner is told: %s", got)
	}
	if page := public(t, internet, http.MethodGet, no, "").Body.String(); !strings.Contains(page, "Another choice was taken") {
		t.Errorf("the other closes: %s", truncate(page))
	}
	if r := public(t, internet, http.MethodPost, no, ""); !strings.Contains(r.Body.String(), "already answered") {
		t.Errorf("and cannot be taken after: %s", r.Body.String())
	}
	if e, _ := a.Store.Get("event", tentative.ID); e.Fields["title"] != "Meeting with Pat Lee" {
		t.Errorf("so the first stands: %v", e.Fields["title"])
	}
}
