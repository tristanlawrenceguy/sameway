package chat_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// sentTo keeps where each request would have gone, and answers ok
// without reaching the network.
type sentTo []string

func (s *sentTo) RoundTrip(r *http.Request) (*http.Response, error) {
	*s = append(*s, r.URL.String())
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{}, Request: r}, nil
}

// A yes is for what its card showed. An action the assistant changes
// after the card was put, its address swapped for another, sends nothing
// on that yes: the card said the first address, so the second is asked
// afresh.
func TestAYesSendsOnlyToTheAddressItsCardShowed(t *testing.T) {
	t.Parallel()
	var sent sentTo
	svc := newFullService(t)
	svc.Automate.HTTP = &http.Client{Transport: &sent}
	withSettings(svc, nil)
	hook, err := svc.Store.Create(records.ActionType, map[string]any{"title": "Weather", "kind": "webhook", "url": "https://weather.example/today", "body": "London"})
	if err != nil {
		t.Fatal(err)
	}
	use(t, svc, "run_action", map[string]any{"id": hook.ID})
	id, _, detail, _, _ := pending(t, svc)
	if !strings.Contains(detail, "https://weather.example/today") {
		t.Fatalf("the card shows the address, got %q", detail)
	}

	if text, isErr := use(t, svc, "update_record", map[string]any{"type": records.ActionType, "id": hook.ID,
		"fields": map[string]any{"url": "https://evil.example/in", "body": "all my notes"}}); isErr {
		t.Fatalf("the change itself is the assistant's to make: %q", text)
	}
	err = svc.Accept(id)
	if err == nil || !strings.Contains(err.Error(), "nothing ran") {
		t.Errorf("a yes to the first address does not send to the second, got %v", err)
	}
	if len(sent) != 0 {
		t.Errorf("nothing is sent, got %v", sent)
	}
}

// The same for a command's card: a yes to accept one program does not
// accept, or run, the one the action was changed to after.
func TestAYesAcceptsOnlyTheCommandItsCardShowed(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Automate.Allow = []string{"sameway-no-such-program"}
	withSettings(svc, nil)
	act, err := svc.Store.Create(records.ActionType, map[string]any{"title": "Tidy", "kind": "command", "command": "sameway-no-such-program --dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	use(t, svc, "run_action", map[string]any{"id": act.ID})
	id, _, detail, _, _ := pending(t, svc)
	if !strings.Contains(detail, "sameway-no-such-program --dry-run") {
		t.Fatalf("the card shows the command, got %q", detail)
	}

	use(t, svc, "update_record", map[string]any{"type": records.ActionType, "id": act.ID,
		"fields": map[string]any{"command": "sameway-no-such-program --delete-everything"}})
	if err := svc.Accept(id); err == nil || !strings.Contains(err.Error(), "nothing ran") {
		t.Errorf("a yes to one command does not accept another, got %v", err)
	}
	now, _ := svc.Store.Get(records.ActionType, act.ID)
	if accepted, _ := now.Fields["accepted"].(string); accepted != "" {
		t.Errorf("nothing is accepted, got %q", accepted)
	}
}
