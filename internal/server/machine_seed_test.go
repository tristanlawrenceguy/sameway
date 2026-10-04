package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// seedLikeAPerson fills a workspace as people and agents do: through the
// API and the assistant, so the log, the receipts and the chat hold what
// they would.
func seedLikeAPerson(t *testing.T, a *app.App, h http.Handler) {
	t.Helper()
	agent := func(method, path string, v any) map[string]any {
		raw, _ := json.Marshal(v)
		req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Go-http-client/1.1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var out map[string]any
		json.Unmarshal(rec.Body.Bytes(), &out)
		if rec.Code >= 300 {
			t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
		}
		return out
	}
	id := func(m map[string]any) string { s, _ := m["id"].(string); return s }
	ana := id(agent(http.MethodPost, "/api/person", map[string]any{"name": "Ana Silva", "email": "ana@example.com"}))
	garden := id(agent(http.MethodPost, "/api/project", map[string]any{"title": "Garden"}))
	paint := id(agent(http.MethodPost, "/api/task", map[string]any{"title": "Buy paint", "status": "doing", "due": "2026-10-09 14:00", "project": garden, "for": ana, "tags": []string{"home"}}))
	agent(http.MethodPatch, "/api/task/"+paint, map[string]any{"status": "done"})
	agent(http.MethodPost, "/api/task", map[string]any{"title": "Water plants", "due": "2026-10-07"})
	agent(http.MethodPost, "/api/note", map[string]any{"title": "Paint colours", "body": "Sage for the hall.", "status": "draft"})
	agent(http.MethodPost, "/api/event", map[string]any{"title": "Pricing", "starts": "2026-10-06 14:00", "people": []string{ana}})
	agent(http.MethodPost, "/api/reminder", map[string]any{"title": "Timesheet", "at": "2026-10-05 09:00", "repeat": "every Monday"})
	habit := id(agent(http.MethodPost, "/api/habit", map[string]any{"name": "Drink water", "cadence": "day", "target": 8, "unit": "glasses"}))
	agent(http.MethodPost, "/api/entry", map[string]any{"habit": habit, "amount": 3, "at": "2026-10-04 08:00"})
	for _, b := range []map[string]any{
		{"component": "collection", "props": map[string]any{"type": "task", "label": "To do", "where": []string{"done=false"}, "order": "due"}},
		{"component": "collection", "props": map[string]any{"type": "note", "label": "Notes", "as": "table"}},
		{"component": "calendar", "props": map[string]any{"type": "event", "caption": "Calendar"}},
		{"component": "chart", "props": map[string]any{"type": "task", "by": "status", "caption": "Tasks by status"}},
		{"component": "tracker", "props": map[string]any{"habits": []string{"Drink water"}}},
	} {
		agent(http.MethodPost, "/api/block", b)
	}

	// A turn whose program failed, as Claude Code's does when signed out.
	a.Chat.Provider, a.Chat.ProviderErr = failing{}, nil
	postForm(t, h, "/chat", map[string][]string{"message": {"Add a task to call the bank"}})
}

type failing struct{}

func (failing) Name() string { return "claude-code" }
func (failing) Complete(context.Context, llm.Request) (*llm.Response, error) {
	return nil, errors.New("claude: exit status 1: Invalid API key · Please run /login")
}
