package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// TestDescribeRoutesContainsMessage verifies that agents can discover the
// message listing surface via GET /api/describe, advancing goal 0010 and
// closing backlog 0223. The "message" key must appear in the routes map so
// an agent calling describe does not have to guess its existence.
func TestDescribeRoutesContainsMessage(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)
	var d struct {
		Routes map[string]string
	}
	decode(t, rec, &d)
	if d.Routes["message"] == "" {
		t.Errorf("routes missing message key — agents cannot discover /api/message via describe")
	} else if !strings.Contains(d.Routes["message"], "GET") || !strings.Contains(d.Routes["message"], "/api/message") {
		t.Errorf("routes.message should contain 'GET /api/message', got %q", d.Routes["message"])
	}
}

// TestMessageEndpointListsRecords verifies GET /api/message actually returns
// the message listing, confirming the endpoint is fully functional and not
// just a stub in the routes map.
func TestMessageEndpointListsRecords(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Chat.Provider, a.Chat.ProviderErr = nil, llm.ErrNotConfigured
	h := server.New(a)

	// Create a couple of messages so there is data to list.
	for i := 1; i <= 2; i++ {
		rec := postJSON(t, h, http.MethodPost, "/api/message", map[string]any{
			"content": fmt.Sprintf("Hello %d", i),
			"role":    "user",
		})
		wantStatus(t, rec, http.StatusCreated)
	}

	rec := get(t, h, "/api/message")
	wantStatus(t, rec, http.StatusOK)
	var msgs struct {
		Count   int
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, rec, &msgs)
	if len(msgs.Records) != 2 {
		t.Errorf("expected 2 messages in listing, got %d", len(msgs.Records))
	}
}
