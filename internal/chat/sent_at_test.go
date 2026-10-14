package chat_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The person's earlier messages reach the model saying when they were
// sent, so "tomorrow" in one is counted from then and an old request is
// not taken for a new one; the message being answered is as they wrote it.
func TestEarlierMessagesSayWhenTheyWereSent(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{{Text: "Noted."}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "remind me to buy milk tomorrow at 5pm"); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(48 * time.Hour)
	svc.Clock = func() time.Time { return later }
	m2 := &scripted{steps: []*llm.Response{{Text: "Here is today."}}}
	svc.Provider = m2
	if _, err := svc.Send(context.Background(), "what is on today?"); err != nil {
		t.Fatal(err)
	}
	var users []string
	for _, msg := range m2.seen[0].Messages {
		if msg.Role == llm.RoleUser {
			users = append(users, msg.Content)
		}
	}
	if len(users) < 2 || !strings.HasPrefix(users[0], "[sent ") || !strings.Contains(users[0], "buy milk tomorrow") {
		t.Errorf("the earlier message says when it was sent: %q", users)
	}
	if last := users[len(users)-1]; last != "what is on today?" {
		t.Errorf("the message being answered is as written: %q", last)
	}
	if !strings.Contains(m2.seen[0].System, "counted from then") {
		t.Error("the model is told a day in an earlier message counts from when it was sent")
	}
}
