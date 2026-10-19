package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person can have several chats. Each has its own history: what was
// said in one is not sent to the model in another. A new chat is named
// after the first thing said in it, and opening an old one brings its
// messages back.
func TestChatsKeepTheirOwnHistory(t *testing.T) {
	t.Parallel()
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
	if n, _ := svc.Store.Count(records.MessageType); n != 2 {
		t.Errorf("deleting a chat takes its messages, leaving the other two, got %d", n)
	}
}

// Messages from before there were several chats name none; they belong to
// the first chat, so nothing said earlier is lost.
func TestOlderMessagesBelongToTheFirstChat(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Provider = &scripted{}
	svc.Store.Create(records.MessageType, map[string]any{"role": "user", "content": "from before"})

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
	t.Parallel()
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
	if n, _ := svc.Store.Count(records.MessageType); n != 2 {
		t.Errorf("the other chat keeps its two messages, got %d", n)
	}
}

// New chat on a chat with nothing said in it keeps that one, so pressing it
// twice, or leaving without writing, does not fill the list with empties.
func TestNewChatOnAnEmptyChatKeepsIt(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	first := svc.Current()
	again, err := svc.NewChat()
	if err != nil {
		t.Fatal(err)
	}
	svc.NewChat()
	if again.ID != first || len(svc.Conversations()) != 1 {
		t.Errorf("an empty chat should be kept, got %d chats", len(svc.Conversations()))
	}
}

// Deleting a chat, or clearing one, can be taken back: the chat and every
// message in it come back as they were.
func TestADeletedOrClearedChatComesBack(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{{Text: "ok"}}}
	if _, err := svc.Send(context.Background(), "plan the garden"); err != nil {
		t.Fatal(err)
	}
	id := svc.Current()
	msgs, _ := svc.MessagesIn(id)
	if err := svc.Clear(); err != nil {
		t.Fatal(err)
	}
	if err := svc.UndoAs("human", ""); err != nil {
		t.Fatalf("a cleared chat should be undoable: %v", err)
	}
	if back, _ := svc.MessagesIn(id); len(back) != len(msgs) {
		t.Errorf("clearing undone should bring back %d messages, got %d", len(msgs), len(back))
	}
	if err := svc.DeleteChat(id); err != nil {
		t.Fatal(err)
	}
	if err := svc.UndoAs("human", ""); err != nil {
		t.Fatalf("a deleted chat should be undoable: %v", err)
	}
	if _, err := svc.Store.Get(records.ConversationType, id); err != nil {
		t.Errorf("the deleted chat should be back")
	}
	if back, _ := svc.MessagesIn(id); len(back) != len(msgs) {
		t.Errorf("its %d messages should be back, got %d", len(msgs), len(back))
	}
}

func contains(s, substr string) bool { return strings.Contains(s, substr) }

// TestNoticeErrorContentIsSanitized: a problem reported through Notice gets its
// technical provider names stripped at storage time, so the stored record
// content never leaks them. This covers backlog 0638 and acceptance item 1.
func TestNoticeErrorContentIsSanitized(t *testing.T) {
	t.Parallel()

	s, _ := newService(t)

	// A real-world error string that contains internal provider names.
	raw := "Your organization has disabled Claude subscription access for Claude Code · Use an Anthropic API key instead, or ask your admin to enable access"

	s.Notice(raw)

	msgs, err := s.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil {
		t.Fatalf("could not read messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected a message after Notice")
	}

	rec := msgs[0]
	content, _ := rec.Fields["content"].(string)

	for _, leak := range []string{"Claude Code", "Anthropic API key"} {
		if contains(content, leak) {
			t.Errorf("Notice(%q): stored content still contains %q (got: %s)", raw, leak, content)
		}
	}
}

// TestNoticeErrorContentStripsProviderPrefixes verifies that provider prefixes
// like "claude:" and "anthropic:" are removed from error content when created
// through Notice. This covers the same path as fail()/failAfter() but via
// a different call site, ensuring all error paths go through SanitizeError.
func TestNoticeErrorContentStripsProviderPrefixes(t *testing.T) {
	t.Parallel()

	s, _ := newService(t)

	raw := "claude: exit status 1: model not found"
	s.Notice(raw)

	msgs, err := s.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil {
		t.Fatalf("could not read messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected a message after Notice")
	}

	rec := msgs[0]
	content, _ := rec.Fields["content"].(string)

	for _, bad := range []string{"claude:", "exit status"} {
		if contains(content, bad) {
			t.Errorf("Notice(%q): stored content still contains %q (got: %s)", raw, bad, content)
		}
	}
}
