package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Someone invited on the Wi-Fi has a name and no email: the owner's
// assistant takes their access away by name, and nobody else's can.
func TestTakingAccessAwayByName(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	if _, err := svc.GiveAccess("Ana", "view"); err != nil {
		t.Fatal(err)
	}
	if text, isErr := use(t, svc, "let_in", map[string]any{"access": "none"}); !isErr || !strings.Contains(text, "their name") {
		t.Errorf("with neither email nor name it asks whose: %q", text)
	}
	text, isErr := use(t, svc, "let_in", map[string]any{"name": "ana", "access": "none"})
	if isErr || !strings.Contains(text, "Ana can now no longer open") {
		t.Fatalf("taken away by name: %q", text)
	}
	if got := svc.PersonByName("Ana").Fields["access"]; got != "" && got != nil {
		t.Errorf("Ana has no access now, has %v", got)
	}
	if text, _ := use(t, svc, "let_in", map[string]any{"name": "Ana", "access": "none"}); !strings.Contains(text, "had no access") {
		t.Errorf("twice says so: %q", text)
	}
}

// An invite on the Wi-Fi and Take away access are written through
// records and logged as whoever did them, and Undo takes each back.
func TestInviteAndTakeAccessUndo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	newest := func() *store.Record {
		t.Helper()
		entries, _ := svc.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
		if len(entries) == 0 {
			t.Fatal("nothing logged")
		}
		return entries[0]
	}
	bob := svc.For(records.Visitor{Name: "Bob", Login: "bob@example.com", Access: records.Host})
	ana, err := bob.GiveAccess("Ana", records.Edit)
	if err != nil {
		t.Fatal(err)
	}
	if e := newest(); e.Fields["action"] != "let in" || e.Fields["by"] != "Bob" {
		t.Errorf("an invite by Bob is logged as %v by %v", e.Fields["action"], e.Fields["by"])
	}
	_, entry, err := svc.TakeAccess(ana.ID)
	if err != nil || entry == "" {
		t.Fatalf("take access: %q %v", entry, err)
	}
	if e := newest(); e.Fields["action"] != "took access from" || (e.Fields["by"] != "" && e.Fields["by"] != nil) {
		t.Errorf("the owner taking access is logged as %v by %v", e.Fields["action"], e.Fields["by"])
	}
	if _, _, err := svc.Undo(entry); err != nil {
		t.Fatal(err)
	}
	if got := svc.PersonByName("Ana").Fields["access"]; got != records.Edit {
		t.Errorf("Undo gives Ana her access back, she has %v", got)
	}
	// Undoing the invite of someone new takes them out again.
	carl, _ := svc.GiveAccess("Carl", records.View)
	if _, _, err := svc.Undo(newest().ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Store.Get(records.PersonType, carl.ID); err == nil {
		t.Error("Undo of a new invite leaves the person")
	}
}
