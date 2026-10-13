package server_test

import (
	"fmt"
	"strings"
	"testing"
)

// TestAChatShowsItsLatestMessages: a long conversation shows its latest
// twenty, newest reply and box to type in near the top of what is read,
// with one quiet link to the earlier ones, saying how many, which opens
// them all in place.
func TestAChatShowsItsLatestMessages(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	for i := 1; i <= 30; i++ {
		if _, err := a.Store.Create("message", map[string]any{"role": "user", "content": fmt.Sprintf("Message number %d.", i)}); err != nil {
			t.Fatal(err)
		}
	}
	page := get(t, h, "/chat").Body.String()
	if strings.Contains(page, "Message number 10.") || !strings.Contains(page, "Message number 11.") || !strings.Contains(page, "Message number 30.") {
		t.Errorf("the chat shows its latest 20 messages, 11 to 30")
	}
	if !strings.Contains(page, ">10 earlier messages<") || !strings.Contains(page, `href="/chat?show=earlier#msg-`) {
		t.Fatalf("a quiet link leads to the earlier messages:\n%s", truncate(page))
	}
	all := get(t, h, "/chat?show=earlier").Body.String()
	if !strings.Contains(all, "Message number 1.") || strings.Contains(all, "earlier messages<") {
		t.Errorf("asked for, the earlier messages are all there, with no link to them")
	}

	b, few := newApp(t)
	for i := 1; i <= 3; i++ {
		b.Store.Create("message", map[string]any{"role": "user", "content": fmt.Sprintf("Short %d.", i)})
	}
	if strings.Contains(get(t, few, "/chat").Body.String(), "earlier message") {
		t.Errorf("a chat of twenty or fewer has nothing earlier to offer")
	}
}
