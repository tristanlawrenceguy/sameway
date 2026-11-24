package automate_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A webhook that runs on its own sends nothing until the owner has
// accepted what it sends, once, with it in front of them: the assistant
// can make one from words it read. Asked once, not every hour; changing
// where it sends asks again.
func TestAWebhookSendsOnItsOwnOnlyOnceAccepted(t *testing.T) {
	t.Parallel()
	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		io.WriteString(w, "ok")
	}))
	defer remote.Close()
	svc, act := newServices(t)
	act.HTTP = remote.Client()
	hook, _ := svc.Store.Create(records.ActionType, map[string]any{"title": "Post tasks", "url": remote.URL, "body": `{"task":"{{title}}"}`, "every": "hour"})
	tuesday := time.Date(2026, 9, 15, 7, 30, 0, 0, time.UTC)

	act.RunDue(context.Background(), tuesday)
	act.RunDue(context.Background(), tuesday.Add(61*time.Minute))
	pending := svc.Proposals()
	if calls != 0 || len(pending) != 1 {
		t.Fatalf("nothing is sent before it is accepted, and it is asked once: %d sent, %d asked", calls, len(pending))
	}
	if detail, _ := pending[0].Fields["detail"].(string); !strings.Contains(detail, remote.URL) || !strings.Contains(detail, "{{title}}") {
		t.Errorf("the question says where it sends and what: %q", detail)
	}

	if err := svc.Accept(pending[0].ID); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Error("accepting does not send: it sends from the next time it is set off")
	}
	act.RunDue(context.Background(), tuesday.Add(122*time.Minute))
	if calls != 1 {
		t.Fatalf("once accepted it sends on its own, got %d", calls)
	}

	svc.Store.Update(records.ActionType, hook.ID, map[string]any{"url": remote.URL + "/elsewhere"})
	act.RunDue(context.Background(), tuesday.Add(183*time.Minute))
	if calls != 1 || len(svc.Proposals()) != 1 {
		t.Errorf("a new address asks again before anything is sent: %d sent, %d asked", calls, len(svc.Proposals()))
	}
}
