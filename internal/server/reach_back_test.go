package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
	a.Chat.Mail = func(to, subject, text string) error { sent <- to + "|" + subject + "|" + text; return nil }

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
