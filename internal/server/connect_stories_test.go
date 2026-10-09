package server_test

import (
	"strings"
	"testing"
)

// With no model, the card asks what the person has and tells each answer's
// story alone: its own steps, its own page, a key field only where a key
// is pasted, and plainly what a ChatGPT subscription cannot do.
func TestConnectingAModelIsAStoryForWhatYouHave(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	page := get(t, h, "/chat").Body.String()
	at := strings.Index(page, "Which of these do you have?")
	if at < 0 {
		t.Fatalf("the card asks what you have: %s", truncate(page))
	}
	card := page[at:]
	for _, want := range []string{"A Claude subscription (Pro or Max)", "free and on this computer", "Anthropic API key", "An OpenRouter key", "A ChatGPT subscription",
		`href="https://console.anthropic.com/settings/keys" target="_blank"`, "cannot be used by other programs"} {
		if !strings.Contains(card, want) {
			t.Errorf("the card has %q", want)
		}
	}
	if n := strings.Count(card, `name="key"`); n != 2 {
		t.Errorf("a key field in the two stories that end in one, not elsewhere: %d", n)
	}
	if strings.Contains(card, "<details open") {
		t.Error("each story is closed until its person opens it")
	}
}
