package server_test

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// quietModel finds no tag for anything, and suggests a task when asked.
type quietModel struct {
	mu    sync.Mutex
	asked []string
}

func (m *quietModel) Name() string { return "quiet" }

func (m *quietModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case strings.Contains(req.System, "You tag a record"):
		m.asked = append(m.asked, req.Messages[0].Content)
		return &llm.Response{Text: `{"tags": []}`}, nil
	case strings.Contains(req.System, "suggest one task"):
		return &llm.Response{Text: `{"make": true, "fields": {"title": "Reply to the school"}, "why": "it asks for a reply"}`}, nil
	}
	return &llm.Response{Text: `{"decide": "keep"}`}, nil
}

// What fits no tag is given nothing to do by Sameway, never asked of the
// model, and folded away on Today as a count; changed to another tag, it
// is a choice the action learns from, and to do sets off the task.
func TestNothingToDoIsFoldedAndTeachesWhatWasMissed(t *testing.T) {
	a, _ := newApp(t)
	model := &quietModel{}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	srv := server.New(a)
	a.Chat.StartAutomating()
	srv.SetUpSorting()

	rec, _ := a.Store.Create("email", map[string]any{"subject": "School newsletter", "body": "Please reply about the trip.", "tags": []any{"to sort"}})
	var cl *store.Record
	tagWait(t, "nothing to do", func() bool {
		cls, _ := a.Store.List(chat.ClassificationType, store.ListOptions{})
		for _, c := range cls {
			if c.Fields["record"] == "email/"+rec.ID {
				cl = c
			}
		}
		return cl != nil
	})
	if cl.Fields["tag"] != "nothing to do" || cl.Fields["why"] != "no other tag fits" {
		t.Fatalf("given by Sameway when no other fits: %v", cl.Fields)
	}
	model.mu.Lock()
	if strings.Contains(strings.Join(model.asked, "|"), "nothing to do") {
		t.Error("the model is never asked about it")
	}
	model.mu.Unlock()

	today := get(t, srv, "/today").Body.String()
	if !strings.Contains(today, "Nothing to do") || strings.Contains(today, "Tags to check") {
		t.Fatalf("folded away, not asked about: %s", truncate(today))
	}
	postForm(t, srv, "/tags/change", url.Values{"id": {cl.ID}, "to": {"to do"}})
	tagWait(t, "changed to to do", func() bool {
		c, _ := a.Store.Get(chat.ClassificationType, cl.ID)
		return c.Fields["state"] == "changed" && c.Fields["to"] == "to do"
	})
	tagWait(t, "a task suggested once it is to do", func() bool {
		for _, p := range a.Chat.Suggestions() {
			if p.Fields["from"] == "email/"+rec.ID {
				return true
			}
		}
		return false
	})
	if page := get(t, srv, "/today").Body.String(); !strings.Contains(page, "Reply to the school") {
		t.Errorf("back among what to sort, with its task: %s", truncate(page))
	}
}
