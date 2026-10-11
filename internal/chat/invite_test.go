package chat_test

import (
	"strings"
	"testing"
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
