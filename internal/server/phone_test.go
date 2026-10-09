package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/notify"
)

// Help offers to send reminders to a phone: one press makes a topic of its
// own, sends a first message there, and keeps it; another stops.
func TestRemindersCanGoToAPhone(t *testing.T) {
	t.Parallel()
	var got []string
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.URL.Path+" "+r.Header.Get("Title")+": "+string(body))
	}))
	defer ntfy.Close()
	a, h := newAppWith(t, app.Options{Ntfy: ntfy.URL})

	if page := get(t, h, "/help").Body.String(); !strings.Contains(page, ">Send reminders to my phone<") || !strings.Contains(page, "pass through ntfy.sh") {
		t.Fatalf("Help offers it, saying where the words go: %s", truncate(page))
	}
	page := after(t, h, postForm(t, h, "/notify/phone", url.Values{"set": {"on"}})).Body.String()
	topic := a.Workspace.Config.Notify.Phone
	if !strings.HasPrefix(topic, ntfy.URL+"/sameway-") || len(topic) < len(ntfy.URL)+30 {
		t.Fatalf("a long topic of its own is kept: %q", topic)
	}
	if len(got) != 1 || !strings.Contains(got[0], "arrive here") || !strings.Contains(page, "subscribe to "+topic) {
		t.Errorf("a first message is sent and the page says how to subscribe: %v %s", got, truncate(page))
	}
	if err := (notify.Notifier{Phone: topic}).Send("Bins", "Take the bins out", "http://127.0.0.1:8080/t/reminder/x"); err != nil || len(got) != 2 || !strings.Contains(got[1], "Bins: Take the bins out") {
		t.Errorf("a ring goes to the phone: %v %v", err, got)
	}
	postForm(t, h, "/notify/phone", url.Values{"set": {"off"}})
	if a.Workspace.Config.Notify.Phone != "" {
		t.Error("and stops")
	}
}
