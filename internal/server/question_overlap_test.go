package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// meanwhile runs something once, the first time it is asked, before it
// answers: another visitor's question, asked while this one is answered.
type meanwhile struct {
	scripted
	then func()
}

func (m *meanwhile) Complete(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if then := m.then; then != nil {
		m.then = nil
		then()
	}
	return m.scripted.Complete(ctx, req)
}

// Two visitors who do not say who they are ask at once. Each is told of
// their own request alone, with its own address, never the other's, and
// the owner hears of each once.
func TestTwoVisitorsAskingAtOnceKeepTheirOwnRequests(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	srv := h.(*server.Server)
	rang := make(chan string, 8)
	srv.OnRing(func(title, text, _ string) { rang <- text })
	internet := srv.Public(nil)
	a.Workspace.Config.Publish.Types = "note"
	ask := func(question string) string {
		req := httptest.NewRequest(http.MethodPost, "/question", strings.NewReader(url.Values{"question": {question}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		internet.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	var second string
	a.Chat.Provider = &meanwhile{scripted: scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Visit from Bea, bea@example.com", "starts": "2026-03-14T11:00:00Z"}}),
		{Text: "Asked."},
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Visit from Al, al@example.com", "starts": "2026-03-14T10:00:00Z"}}),
		{Text: "Asked."},
	}}, then: func() { second = ask("Can Bea visit at 11?") }}

	first := ask("Can Al visit at 10?")
	if !strings.Contains(first, "Visit from Al") || strings.Contains(first, "Visit from Bea") {
		t.Errorf("the first is told of their own request alone:\n%s", truncate(first))
	}
	if !strings.Contains(second, "Visit from Bea") || strings.Contains(second, "Visit from Al") {
		t.Errorf("the second too:\n%s", truncate(second))
	}
	// The owner is told in the background: two are waited for, then a
	// moment more for any third.
	heard := 0
	for _, wait := range []time.Duration{5 * time.Second, 5 * time.Second, 300 * time.Millisecond} {
		select {
		case <-rang:
			heard++
		case <-time.After(wait):
		}
	}
	if heard != 2 {
		t.Errorf("the owner hears of each request once: %d times", heard)
	}
}
