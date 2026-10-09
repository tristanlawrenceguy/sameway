package server_test

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// sortModel answers what sorting asks: the tag, the task, the check.
type sortModel struct {
	mu        sync.Mutex
	suggested []string // what each suggestion was asked, in order
}

func (m *sortModel) Name() string { return "sort" }

func (m *sortModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case strings.Contains(req.System, "You tag a record"):
		return &llm.Response{Text: `{"tags": [{"tag": "to do", "why": "asks for a payment"}]}`}, nil
	case strings.Contains(req.System, "suggest one task"):
		m.suggested = append(m.suggested, req.Messages[0].Content)
		return &llm.Response{Text: `{"make": true, "fields": {"title": "Pay the bill", "due": "by Friday"}, "why": "a bill to pay"}`}, nil
	}
	return &llm.Response{Text: `{"decide": "keep"}`}, nil
}

// An email coming in is tagged, and an email tagged to do sets off the
// suggestion of a task, filled from it with its day counted from when it
// was sent; Today puts it beside the email, made, changed or turned down
// with a press, which sorts the email; the next suggestion is shown what
// the person did with the last ones.
func TestAnEmailToDoIsSuggestedAsATask(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	model := &sortModel{}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	srv := server.New(a)
	a.Chat.StartAutomating()
	srv.SetUpSorting()

	waiting := func(subject string) *store.Record {
		var found *store.Record
		tagWait(t, "a suggestion from "+subject, func() bool {
			for _, p := range a.Chat.Suggestions() {
				if strings.Contains(p.Fields["summary"].(string), subject) {
					found = p
				}
			}
			return found != nil
		})
		return found
	}
	arrive := func(subject string) *store.Record {
		rec, err := a.Store.Create("email", map[string]any{"subject": subject, "from": "Water Co", "received": "2026-10-12T09:00:00Z", "body": "Please pay by Friday.", "tags": []any{"to sort"}})
		if err != nil {
			t.Fatal(err)
		}
		return rec
	}

	first := arrive("Water bill")
	p := waiting("Water bill")
	if s := p.Fields["summary"].(string); !strings.Contains(s, `Make a task from "Water bill": Pay the bill, due`) || !strings.Contains(s, "16 Oct") {
		t.Errorf("the task, its day counted from when the email was sent: %s", s)
	}
	today := get(t, srv, "/today").Body.String()
	at := strings.Index(today, "To sort")
	if at < 0 || !strings.Contains(today[at:], "Make a task from") || !strings.Contains(today[at:], ">Make it<") {
		t.Fatalf("Today puts it beside the email: %s", truncate(today))
	}
	postForm(t, srv, "/suggested/yes", url.Values{"id": {p.ID}})
	done, _ := a.Store.Get(records.ProposalType, p.ID)
	made, _ := done.Fields["made"].(string)
	if done.Fields["state"] != "accepted" || !strings.HasPrefix(made, "task/") {
		t.Fatalf("made, and what it made kept: %v", done.Fields)
	}
	if got, _ := a.Store.Get("email", first.ID); strings.Contains(strings.Join(anyStrings(got.Fields["tags"]), ","), "to sort") {
		t.Error("the email is sorted")
	}
	a.Store.Update("task", strings.TrimPrefix(made, "task/"), map[string]any{"title": "Pay the water bill"})

	arrive("Gas bill")
	p = waiting("Gas bill")
	postForm(t, srv, "/suggested/no", url.Values{"id": {p.ID}})

	arrive("Phone bill")
	waiting("Phone bill")
	model.mu.Lock()
	last := model.suggested[len(model.suggested)-1]
	model.mu.Unlock()
	if !strings.Contains(last, "turned it down") || !strings.Contains(last, "then changed title from Pay the bill to Pay the water bill") {
		t.Errorf("the next suggestion is shown what the person did with the last ones: %s", last)
	}
	if strings.Index(last, "Gas bill") > strings.Index(last, "Water bill") {
		t.Error("newest first")
	}
}
