package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

var bob = chat.Visitor{Name: "Bob", Login: "Bob@Example.com", Access: chat.Edit, Device: "pixel-7"}

// Bob has his own conversations with the assistant: he sees none of the
// owner's, the owner sees none of his, and he cannot open the owner's.
func TestEachPersonHasTheirOwnChats(t *testing.T) {
	svc := newFullService(t)
	svc.Say("the owner's words")
	ownerChat := svc.Current()

	b := svc.For(bob)
	if b.Current() == ownerChat {
		t.Fatal("Bob's chat must not be the owner's")
	}
	b.Say("Bob's words")
	if msgs, _ := b.Messages(); len(msgs) != 1 || !strings.Contains(msgs[0].Fields["content"].(string), "Bob's") {
		t.Errorf("Bob sees only his own words: %v", msgs)
	}
	if msgs, _ := svc.Messages(); len(msgs) != 1 || !strings.Contains(msgs[0].Fields["content"].(string), "owner's") {
		t.Errorf("the owner sees only their own words: %v", msgs)
	}
	if err := b.OpenChat(ownerChat); err == nil {
		t.Error("Bob cannot open the owner's chat")
	}
	if len(b.Conversations()) != 1 || len(svc.Conversations()) != 1 {
		t.Errorf("each has one chat: Bob %d, owner %d", len(b.Conversations()), len(svc.Conversations()))
	}
}

// What only the owner may have done is neither offered to Bob's assistant
// nor done if it tries, and it knows whom it is talking to.
func TestSomeoneElsesAssistantKeepsToTheirAccess(t *testing.T) {
	svc := newFullService(t)
	cfg := withSettings(svc, map[string]string{"ui.pace": "calm"})
	b := svc.For(bob)
	for _, tool := range b.Tools() {
		if tool.Name == "set_setting" || tool.Name == "undo_change" || tool.Name == "update_sameway" {
			t.Errorf("%s should not be offered to Bob", tool.Name)
		}
	}
	text, isErr := use(t, b, "set_setting", map[string]any{"key": "ui.pace", "value": "still"})
	if !isErr || cfg["ui.pace"] != "calm" {
		t.Errorf("Bob's assistant cannot change a setting: %q", text)
	}
	if text, isErr := use(t, b, "let_in", map[string]any{"email": "carol@example.com", "access": "none"}); !isErr {
		t.Errorf("only the owner takes access away: %q", text)
	}

	m := &scripted{}
	b.Provider = m
	b.Send(context.Background(), "hi")
	if len(m.seen) == 0 || !strings.Contains(m.seen[0].System, "You are talking with Bob") {
		t.Error("the assistant should be told it is talking with Bob")
	}
}

// When the owner lets someone in, the chat says the one step that is
// theirs in Tailscale: sharing this computer with them.
func TestLettingSomeoneInSaysHowToShareTheMachine(t *testing.T) {
	svc := newFullService(t)
	withSettings(svc, map[string]string{"tailnet.name": "home"})
	use(t, svc, "let_in", map[string]any{"email": "carol@example.com", "name": "Carol", "access": "view"})
	id, _, _, _, _ := pending(t, svc)
	if err := svc.Accept(id); err != nil {
		t.Fatal(err)
	}
	msgs, _ := svc.Messages()
	found := false
	for _, m := range msgs {
		if c, _ := m.Fields["content"].(string); strings.Contains(c, "share this computer (home) with carol@example.com") && strings.Contains(c, "https://login.tailscale.com/admin/machines") {
			found = true
		}
	}
	if !found {
		t.Error("the chat should say how to share the machine with Carol")
	}
}
