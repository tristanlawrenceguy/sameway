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
// conversation; and a reply is read with what came before it.
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

	m := &heardModel{}
	a.Chat.Provider, a.Chat.ProviderErr = m, nil
	act, _ := a.Store.Create("action", map[string]any{"title": "Sort", "kind": "classify"})
	a.Store.Create("tag", map[string]any{"name": "to do", "means": "Something I have to do."})
	_ = a.Chat.Classify(context.Background(), act, "email", thanks.ID)
	m.mu.Lock()
	q := m.asked[len(m.asked)-1]
	m.mu.Unlock()
	if !strings.Contains(q, "This record is the latest message") || !strings.Contains(q, "Can you send the quote") || !strings.Contains(q, "from the person themselves: Here it is") {
		t.Errorf("a reply is read with what came before it, saying who wrote it: %s", q)
	}
}
