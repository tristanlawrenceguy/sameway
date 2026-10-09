package server_test

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/mailin/mailintest"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func message(id, refs, from, to, subject, body string) string {
	date := map[string]string{"q1@joe.example": "09", "r1@me.example": "10", "q2@joe.example": "11"}[id]
	if date == "" {
		date = "12"
	}
	head := "Message-Id: <" + id + ">\r\nDate: Mon, 12 Oct 2026 " + date + ":00:00 +0000\r\nFrom: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n"
	if refs != "" {
		head += "In-Reply-To: <" + refs + ">\r\nReferences: <" + refs + ">\r\n"
	}
	return head + "Content-Type: text/plain\r\n\r\n" + body + "\r\n"
}

// heardModel keeps each tagging question.
type heardModel struct {
	mu    sync.Mutex
	asked []string
}

func (m *heardModel) Name() string { return "heard" }
func (m *heardModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asked = append(m.asked, req.Messages[0].Content)
	return &llm.Response{Text: `{"tags": []}`}, nil
}

// An email answering another is tied to the conversation it is in by what
// it says it answers; the person's own reply, from their Sent folder, is
// part of it and says they sent it; the first email's page lists the
// conversation; and its latest email says whose turn it is, on Today too.
func TestAnEmailConversationIsAThread(t *testing.T) {
	t.Setenv("SAMEWAY_KEYS", filepath.Join(t.TempDir(), "keys.json"))
	server.MailInsecure(true)
	defer server.MailInsecure(false)
	mail := mailintest.Start(t, "me@example.com", "pw")
	mail.Folder("Sent")
	mail.Put(t, "INBOX", message("q1@joe.example", "", "Joe <joe@joe.example>", "me+sameway@example.com", "Quote", "Can you send the quote by Friday?"))
	mail.Put(t, "Sent", message("r1@me.example", "q1@joe.example", "Me <me@example.com>", "joe@joe.example", "Re: Quote", "Here it is, attached."))
	mail.Put(t, "Sent", message("other@me.example", "", "Me <me@example.com>", "ann@x.example", "Lunch", "Lunch Tuesday?"))
	mail.Put(t, "INBOX", message("q2@joe.example", "r1@me.example", "Joe <joe@joe.example>", "me+sameway@example.com", "Re: Quote", "Thanks, got it!"))
	mail.Put(t, "INBOX", message("w1@ann.example", "", "Ann <ann@ann.example>", "me+sameway@example.com", "Plans", "Dinner soon."))
	mail.Put(t, "Sent", message("w2@me.example", "w1@ann.example", "Me <me@example.com>", "ann@ann.example", "Re: Plans", "Yes! Which day suits you?"))
	mail.Put(t, "INBOX", message("v1@bo.example", "", "Bo <bo@bo.example>", "me+sameway@example.com", "Room", "I booked the room."))
	mail.Put(t, "INBOX", message("v2@bo.example", "v1@bo.example", "Bo <bo@bo.example>", "me+sameway@example.com", "Re: Room", "The big one or the small one?"))
	a, h := newApp(t)
	postForm(t, h, "/mail/connect", url.Values{"user": {"me@example.com"}, "password": {"pw"}, "host": {mail.Addr}})

	emails, _ := a.Store.List("email", store.ListOptions{OrderBy: "created_at"})
	byBody := map[string]*store.Record{}
	for _, e := range emails {
		byBody[e.Fields["body"].(string)] = e
	}
	first, mine, thanks := byBody["Can you send the quote by Friday?"], byBody["Here it is, attached."], byBody["Thanks, got it!"]
	if first == nil || thanks == nil {
		t.Fatalf("the conversation came in: %d emails", len(emails))
	}
	if byBody["Lunch Tuesday?"] != nil {
		t.Error("what the person sent outside the workspace's conversations is not read")
	}
	if thanks.Fields["thread"] != first.ID {
		t.Errorf("the reply is in the first email's thread: %v", thanks.Fields["thread"])
	}
	if mine == nil || mine.Fields["from_me"] != true || mine.Fields["thread"] != first.ID || strings.Contains(strings.Join(anyStrings(mine.Fields["tags"]), ","), "to sort") {
		t.Errorf("the person's own reply is in it, as theirs, not to sort: %v", mine)
	}
	page := get(t, h, "/t/email/"+first.ID+"?show=points-here:email.thread").Body.String()
	if !strings.Contains(page, "Thanks, got it!") && !strings.Contains(page, "Re: Quote") {
		t.Errorf("the first email's page lists the conversation: %s", truncate(page))
	}

	turns := map[string]any{}
	for _, e := range func() []*store.Record { r, _ := a.Store.List("email", store.ListOptions{}); return r }() {
		turns[e.Fields["body"].(string)] = e.Fields["turn"]
	}
	if turns["Thanks, got it!"] != "" && turns["Thanks, got it!"] != nil || turns["Here it is, attached."] != "" && turns["Here it is, attached."] != nil {
		t.Errorf("a conversation closed with thanks owes nothing, and only its latest says: %v", turns)
	}
	if turns["Yes! Which day suits you?"] != "theirs" || turns["The big one or the small one?"] != "yours" {
		t.Errorf("whose turn: you wrote last, theirs; they asked last, yours: %v", turns)
	}
	today := get(t, h, "/today").Body.String()
	if !strings.Contains(today, "Your turn to reply") || !strings.Contains(today, "The big one or the small one?") || !strings.Contains(today, "Waiting on them") {
		t.Errorf("Today says whose turn: %s", truncate(today))
	}
}
