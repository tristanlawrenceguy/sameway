package chat_test

import (
	"strings"
	"testing"
)

// Letting someone in is put to the owner in plain words, and a yes makes
// them a person with that access; taking it away is not asked. Nobody's
// access changes before the owner answers.
func TestLettingSomeoneInIsAskedAndTakingItBackIsNot(t *testing.T) {
	svc := newFullService(t)
	text, isErr := use(t, svc, "let_in", map[string]any{"email": "Bob@Example.com", "name": "Bob", "access": "edit"})
	if isErr || !strings.Contains(text, "asked the person") {
		t.Fatalf("giving access is asked: %q", text)
	}
	if svc.PersonByEmail("bob@example.com") != nil {
		t.Fatal("nobody is let in before the owner answers")
	}
	id, ask, detail, yes, _ := pending(t, svc)
	if ask != "Let Bob (bob@example.com) edit this workspace?" || !strings.Contains(detail, "signed in to Tailscale as bob@example.com") || !strings.Contains(detail, "not its settings") || yes != "Yes, let them edit" {
		t.Errorf("the question says who, how and what they could do: %q / %q / %q", ask, detail, yes)
	}
	if err := svc.Accept(id); err != nil {
		t.Fatal(err)
	}
	bob := svc.PersonByEmail("bob@example.com")
	if bob == nil || bob.Fields["access"] != "edit" || bob.Fields["name"] != "Bob" {
		t.Fatalf("a yes lets Bob edit: %+v", bob)
	}

	text, isErr = use(t, svc, "let_in", map[string]any{"email": "bob@example.com", "access": "none"})
	if isErr || strings.Contains(text, "asked the person") {
		t.Fatalf("taking access away is simply done: %q", text)
	}
	if got := svc.PersonByEmail("bob@example.com"); got.Fields["access"] != "" && got.Fields["access"] != nil {
		t.Errorf("Bob should have no access now, has %v", got.Fields["access"])
	}
}

// Hosting a copy is the owner's gravest yes, and the question says so.
func TestLettingSomeoneHostSaysWhatThatMeans(t *testing.T) {
	svc := newFullService(t)
	use(t, svc, "let_in", map[string]any{"email": "hana@example.com", "name": "Hana", "access": "host"})
	_, ask, detail, yes, _ := pending(t, svc)
	if ask != "Let Hana (hana@example.com) host this workspace too?" || !strings.Contains(detail, "full copy") || !strings.Contains(detail, "trust") || yes != "Yes, let them host it" {
		t.Errorf("the question says what hosting gives them: %q / %q / %q", ask, detail, yes)
	}
}
