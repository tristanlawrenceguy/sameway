package server_test

import (
	"strings"
	"testing"
)

// Connecting email asks for the address, then tells that service's story
// alone: its own page, opened in a new tab, and nothing of the others; a
// service that cannot be read says so and its way round.
func TestEachMailServiceHasItsOwnStory(t *testing.T) {
	_, h := newApp(t)
	first := get(t, h, "/mail").Body.String()
	if !strings.Contains(first, "which email do you use") || strings.Contains(first, "Gmail") || strings.Contains(first, "iCloud") {
		t.Errorf("the address first, no list of services: %s", truncate(first))
	}
	gmail := get(t, h, "/mail?user=me@gmail.com").Body.String()
	for _, want := range []string{`href="https://myaccount.google.com/apppasswords" target="_blank"`, "16 letters", "Connect Gmail", `value="me@gmail.com"`} {
		if !strings.Contains(gmail, want) {
			t.Errorf("Gmail's story (%s): %s", want, truncate(gmail))
		}
	}
	if strings.Contains(gmail, "Apple") || strings.Contains(gmail, "Yahoo") || strings.Contains(gmail, "Mail server") {
		t.Error("only Gmail's story")
	}
	if icloud := get(t, h, "/mail?user=me@icloud.com").Body.String(); !strings.Contains(icloud, "App-Specific Passwords") || strings.Contains(icloud, "16 letters") {
		t.Errorf("iCloud's own story: %s", truncate(icloud))
	}
	if outlook := get(t, h, "/mail?user=me@outlook.com").Body.String(); !strings.Contains(outlook, "no longer lets programs read mail") || strings.Contains(outlook, `name="password"`) {
		t.Errorf("Outlook says plainly it cannot, and its way round: %s", truncate(outlook))
	}
	if other := get(t, h, "/mail?user=me@example.org").Body.String(); !strings.Contains(other, `value="imap.example.org:993"`) {
		t.Errorf("an unknown service is asked its server, guessed: %s", truncate(other))
	}
}
