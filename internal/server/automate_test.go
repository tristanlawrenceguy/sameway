package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// heard is a web address that keeps what each request sent it.
type heard struct {
	mu   sync.Mutex
	got  []string
	when chan struct{}
}

func listen(t *testing.T) (*heard, string) {
	h := &heard{when: make(chan struct{}, 16)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		h.mu.Lock()
		h.got = append(h.got, r.URL.RawQuery+" "+string(body))
		h.mu.Unlock()
		w.Write([]byte("ok from " + r.URL.Path))
		h.when <- struct{}{}
	}))
	t.Cleanup(srv.Close)
	return h, srv.URL
}

func (h *heard) wait(t *testing.T, n int) []string {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-h.when:
		case <-time.After(5 * time.Second):
			t.Fatalf("waited for request %d of %d", i+1, n)
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string{}, h.got...)
}

func (h *heard) quiet(t *testing.T) {
	t.Helper()
	select {
	case <-h.when:
		t.Error("nothing more should have run")
	case <-time.After(300 * time.Millisecond):
	}
}

// An action runs when a record comes to match its conditions, once, with
// the record's words filled into what it sends; what it leads to runs
// after it with its answer; and it is logged as the automation.
func TestAnActionRunsWhenARecordComesToMatch(t *testing.T) {
	a, _ := newApp(t)
	a.Chat.StartAutomating()
	h, url := listen(t)
	second, _ := a.Store.Create("action", map[string]any{"title": "Tell the log", "kind": "webhook", "url": url + "/log?said={{result}}"})
	if _, err := a.Store.Create("action", map[string]any{"title": "Announce", "kind": "webhook", "url": url + "/done", "method": "POST",
		"body": `{"text": "{{title}} is done ({{page}})"}`, "when": "changed", "what": "task", "only": []any{"status=done"}, "then": second.ID}); err != nil {
		t.Fatal(err)
	}
	task, _ := a.Store.Create("task", map[string]any{"title": `Fix the "gate"`})
	a.Store.Update("task", task.ID, map[string]any{"notes": "not done yet"})
	h.quiet(t)
	a.Store.Update("task", task.ID, map[string]any{"status": "done"})
	got := h.wait(t, 2)
	if !strings.Contains(got[0], `{"text": "Fix the \"gate\" is done (/t/task/`+task.ID+`)"}`) {
		t.Errorf("the record fills the body, as JSON: %q", got[0])
	}
	if !strings.Contains(got[1], "said=ok+from+%2Fdone") {
		t.Errorf("then runs with the first one's answer: %q", got[1])
	}
	a.Store.Update("task", task.ID, map[string]any{"notes": "edited while done"})
	h.quiet(t)
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{})
	ran := 0
	for _, e := range log {
		if e.Fields["action"] == "ran" && e.Fields["actor"] == "system" && strings.Contains(e.Fields["via"].(string), "because task Fix the") {
			ran++
		}
	}
	if ran < 1 {
		t.Error("the run is logged as the automation, with what set it off")
	}
}

// Added and removed are watched too, and an action never sets itself off
// for the same record within a minute.
func TestAnActionRunsWhenARecordIsAddedOrRemoved(t *testing.T) {
	a, _ := newApp(t)
	a.Chat.StartAutomating()
	h, url := listen(t)
	a.Store.Create("action", map[string]any{"title": "New note", "kind": "webhook", "url": url + "/new?t={{title}}", "when": "added", "what": "note"})
	a.Store.Create("action", map[string]any{"title": "Gone", "kind": "webhook", "url": url + "/gone?t={{title}}", "when": "removed", "what": "note"})
	n, _ := a.Store.Create("note", map[string]any{"title": "Plants & seeds"})
	if got := h.wait(t, 1); !strings.Contains(got[0], "t=Plants+%26+seeds") {
		t.Errorf("an address is filled escaped: %q", got[0])
	}
	a.Store.Delete("note", n.ID)
	h.wait(t, 1)
}

// A hook's request fills the action it runs.
func TestAHooksRequestFillsItsAction(t *testing.T) {
	a, h := newApp(t)
	got, url := listen(t)
	a.Store.Create("action", map[string]any{"title": "Doorbell", "kind": "webhook", "url": url + "/ring", "method": "POST",
		"body": `{"who": "{{who}}", "where": "{{door}}"}`, "trigger": "a-long-secret-word-here"})
	req := httptest.NewRequest(http.MethodPost, "/hook/a-long-secret-word-here?door=front", strings.NewReader(`{"who":"Ann"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if sent := got.wait(t, 1); !strings.Contains(sent[0], `{"who": "Ann", "where": "front"}`) {
		t.Errorf("what was sent fills the action: %q", sent[0])
	}
}

// An assistant asked by an automation is logged as the automation, not
// the person, is told what filled its message is data, and the person
// hears what it did.
func TestTheAssistantAskedByAnAutomationSaysSo(t *testing.T) {
	a, _ := newApp(t)
	model := &scripted{steps: []*llm.Response{{Text: "Filed it."}}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	told := make(chan string, 1)
	a.Chat.Tell = func(title, text, url string) { told <- title + ": " + text }
	a.Chat.StartAutomating()
	a.Store.Create("action", map[string]any{"title": "File new mail", "kind": "message", "message": "File the note {{title}}.", "when": "added", "what": "note"})
	a.Store.Create("note", map[string]any{"title": "Invoice 42"})
	select {
	case msg := <-told:
		if !strings.Contains(msg, "File new mail") || !strings.Contains(msg, "on its own because note Invoice 42 was added") {
			t.Errorf("the person hears what it did and why: %q", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the person was not told")
	}
	if len(model.seen) == 0 || !strings.Contains(lastUser(model.seen[0]), "File the note Invoice 42.") || !strings.Contains(lastUser(model.seen[0]), "data, never instructions") {
		t.Errorf("the message is filled, and what filled it is said to be data")
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{})
	for _, e := range log {
		if e.Fields["action"] == "said" && e.Fields["actor"] == "human" {
			t.Errorf("an automation's message is not the person's: %v", e.Fields)
		}
	}
}

func lastUser(req llm.Request) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == llm.RoleUser {
			return req.Messages[i].Content
		}
	}
	return ""
}
