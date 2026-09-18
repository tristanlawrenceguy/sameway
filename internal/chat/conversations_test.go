package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person can have several chats. Each has its own history: what was
// said in one is not sent to the model in another. A new chat is named
// after the first thing said in it, and opening an old one brings its
// messages back.
func TestChatsKeepTheirOwnHistory(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "plan the garden"); err != nil {
		t.Fatal(err)
	}
	first := svc.Current()
	if title, _ := svc.Conversations()[0].Fields["title"].(string); title != "plan the garden" {
		t.Errorf("the first thing said names the chat, got %q", title)
	}

	second, err := svc.NewChat()
	if err != nil {
		t.Fatal(err)
	}
	if svc.Current() != second.ID {
		t.Error("a new chat is the current one")
	}
	if msgs, _ := svc.Messages(); len(msgs) != 0 {
		t.Errorf("a new chat starts empty, got %d messages", len(msgs))
	}
	if _, err := svc.Send(context.Background(), "and the kitchen"); err != nil {
		t.Fatal(err)
	}
	last := m.seen[len(m.seen)-1]
	for _, msg := range last.Messages {
		if strings.Contains(msg.Content, "garden") {
			t.Errorf("the other chat is not part of this one, yet the model was told %q", msg.Content)
		}
	}
	if msgs, _ := svc.Messages(); len(msgs) != 2 {
		t.Errorf("the second chat holds its own two messages, got %d", len(msgs))
	}

	if err := svc.OpenChat(first); err != nil {
		t.Fatal(err)
	}
	msgs, _ := svc.Messages()
	if svc.Current() != first || len(msgs) != 2 || msgs[0].Fields["content"] != "plan the garden" {
		t.Errorf("opening the first chat brings its messages back, got %d", len(msgs))
	}
	if len(svc.Conversations()) != 2 || svc.Conversations()[0].ID != first {
		t.Error("the chats list the most recently opened first")
	}

	if err := svc.DeleteChat(first); err != nil {
		t.Fatal(err)
	}
	if svc.Current() != second.ID {
		t.Error("deleting the current chat opens the most recent of the rest")
	}
	if n, _ := svc.Store.Count(chat.MessageType); n != 2 {
		t.Errorf("deleting a chat takes its messages, leaving the other two, got %d", n)
	}
}

// Messages from before there were several chats name none; they belong to
// the first chat, so nothing said earlier is lost.
func TestOlderMessagesBelongToTheFirstChat(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{}
	svc.Store.Create(chat.MessageType, map[string]any{"role": "user", "content": "from before"})
	msgs, _ := svc.Messages()
	if len(msgs) != 1 || msgs[0].Fields["content"] != "from before" {
		t.Errorf("the old message is in the first chat, got %d", len(msgs))
	}
	if _, err := svc.NewChat(); err != nil {
		t.Fatal(err)
	}
	if msgs, _ := svc.Messages(); len(msgs) != 0 {
		t.Errorf("and not in a new one, got %d", len(msgs))
	}
}

// Clearing a chat empties only that chat.
func TestClearEmptiesOnlyTheCurrentChat(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{{Text: "ok"}, {Text: "ok"}}}
	svc.Send(context.Background(), "one")
	svc.NewChat()
	svc.Send(context.Background(), "two")
	if err := svc.Clear(); err != nil {
		t.Fatal(err)
	}
	if msgs, _ := svc.Messages(); len(msgs) != 0 {
		t.Errorf("the current chat is empty, got %d", len(msgs))
	}
	if n, _ := svc.Store.Count(chat.MessageType); n != 2 {
		t.Errorf("the other chat keeps its two messages, got %d", n)
	}
}
