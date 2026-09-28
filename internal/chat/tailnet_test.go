package chat_test

import (
	"strings"
	"testing"
	"time"
)

// Opening the workspace to the person's phone reaches out to Tailscale and
// lets other devices in, so it is asked first, in their words. A yes turns
// it on and the next step (signing in) is in the chat for them to follow;
// turning it off sends nothing anywhere and is not asked.
func TestOpeningToThePhoneIsAskedAndTheNextStepIsInTheChat(t *testing.T) {
	svc := newFullService(t)
	cfg := withSettings(svc, nil)
	svc.Tailnet = func(time.Duration) string {
		svc.Say("To open this workspace from your phone, sign in to Tailscale here: https://login.tailscale.com/a/abc")
		return "To open this workspace from your phone, sign in to Tailscale here."
	}

	text, isErr := use(t, svc, "set_setting", map[string]any{"key": "tailnet.name", "value": "home"})
	if isErr || !strings.Contains(text, "asked the person") {
		t.Fatalf("turning it on is asked, not done: %q", text)
	}
	if cfg["tailnet.name"] != "" {
		t.Fatal("nothing changes before the person answers")
	}
	id, ask, detail, yes, no := pending(t, svc)
	if ask != "Open this workspace from your phone?" || !strings.Contains(detail, "Nobody else can") || !strings.Contains(detail, "Tailscale app") {
		t.Errorf("the question says what opens to whom and what it needs, got %q / %q", ask, detail)
	}
	if strings.Contains(ask+detail, "tailnet.name") || yes == "" || no == "" {
		t.Errorf("the question is in the person's words with both answers: %q %q %q", ask+detail, yes, no)
	}
	if err := svc.Accept(id); err != nil {
		t.Fatal(err)
	}
	if cfg["tailnet.name"] != "home" {
		t.Error("a yes turns it on under the name asked about")
	}
	msgs, err := svc.Messages()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range msgs {
		if c, _ := m.Fields["content"].(string); m.Fields["role"] == "assistant" && strings.Contains(c, "https://login.tailscale.com/a/abc") {
			found = true
		}
	}
	if !found {
		t.Error("the sign-in link should be in the chat after the yes")
	}

	text, isErr = use(t, svc, "set_setting", map[string]any{"key": "tailnet.name", "value": ""})
	if isErr || strings.Contains(text, "asked the person") || cfg["tailnet.name"] != "" {
		t.Errorf("turning it off is simply done: %q", text)
	}
}
