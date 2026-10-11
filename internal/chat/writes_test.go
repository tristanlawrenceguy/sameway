package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What someone let in does to their own chats is logged as theirs, by
// name, not as the owner's: the chat writes through records and logs as
// the visitor it speaks for, where it used to log every such change as a
// bare "human", which reads as "You".
func TestChatChangesAreLoggedAsTheVisitor(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	bob := svc.For(records.Visitor{Name: "Bob", Login: "bob@example.com", Access: records.Edit})
	logged := func(action string) *store.Record {
		t.Helper()
		entries, _ := svc.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
		for _, e := range entries {
			if e.Fields["action"] == action && e.Fields["target"] == "conversation" {
				return e
			}
		}
		t.Fatalf("nothing %s in the log", action)
		return nil
	}

	chat, err := bob.NewChat()
	if err != nil {
		t.Fatal(err)
	}
	bob.Say("Hello Bob")
	if msgs, _ := bob.MessagesIn(chat.ID); len(msgs) != 1 {
		t.Fatalf("Bob's chat has %d messages, want 1", len(msgs))
	}
	if err := bob.Clear(); err != nil {
		t.Fatal(err)
	}
	if e := logged("cleared"); e.Fields["by"] != "Bob" || e.Fields["actor"] != "human" {
		t.Errorf("a chat cleared by Bob is logged by %v (%v), want Bob", e.Fields["by"], e.Fields["actor"])
	}
	bob.Say("Again")
	if err := bob.DeleteChat(chat.ID); err != nil {
		t.Fatal(err)
	}
	e := logged("deleted")
	if e.Fields["by"] != "Bob" {
		t.Errorf("a chat deleted by Bob is logged by %q, want Bob", e.Fields["by"])
	}
	if !svc.Undoable(e) {
		t.Error("a chat deleted cannot be undone")
	}
	if left, _ := svc.Store.List(records.MessageType, store.ListOptions{}); len(left) != 0 {
		t.Errorf("deleting the chat left %d messages", len(left))
	}

	// The owner's own are the owner's: logged with no name, as "You".
	mine, _ := svc.NewChat()
	if err := svc.DeleteChat(mine.ID); err != nil {
		t.Fatal(err)
	}
	if e := logged("deleted"); e.Fields["by"] != "" && e.Fields["by"] != nil {
		t.Errorf("the owner's chat deleted is logged by %q, want nobody named", e.Fields["by"])
	}
}
