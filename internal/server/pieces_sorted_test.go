package server_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// seesPieces says the second part of a letter continues the first, and
// that a gym timetable continues nothing.
type seesPieces struct {
	mu    sync.Mutex
	asked []string
}

func (m *seesPieces) Name() string { return "pieces" }
func (m *seesPieces) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := req.Messages[0].Content
	if strings.Contains(req.System, "one at a time") {
		m.asked = append(m.asked, q)
		now, _, _ := strings.Cut(q, "Shared before it")
		if strings.Contains(now, "part 2") {
			return &llm.Response{Text: `{"continues": 1, "why": "the rest of the same letter"}`}, nil
		}
		return &llm.Response{Text: `{"continues": 0}`}, nil
	}
	return &llm.Response{Text: `{"tags": []}`}, nil
}

// A letter shared in two parts, one after the other, is one note: the
// first part's words come first, it keeps the first part's name, the
// first note goes, and what was suggested from the first part alone is
// put away to be asked again from the whole. Something else shared
// after stays apart, and a share with nothing before it asks nothing.
func TestPiecesSharedOneAfterAnotherAreOne(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &seesPieces{}
	a.Chat.Provider = model
	var tick atomic.Int64 // each share a minute after the one before
	start := time.Now()
	a.Store.Clock = func() time.Time { return start.Add(time.Duration(tick.Add(1)) * time.Minute) }
	share := func(title, text string) *store.Record {
		form := url.Values{"title": {title}, "text": {text}}.Encode()
		if rec := do(t, h, http.MethodPost, "/share", strings.NewReader(form), "application/x-www-form-urlencoded"); rec.Code != http.StatusSeeOther {
			t.Fatalf("shared: %d", rec.Code)
		}
		note := lastNote(t, a)
		act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort", "kind": "classify", "tags": []any{"to do"}})
		a.Automate.Classify(context.Background(), act, "note", note.ID)
		got, _ := a.Store.Get("note", note.ID)
		return got
	}
	first := share("Lease renewal, 14 Mill Lane", "Lease renewal part 1: the rent rises to 1,250 a month from January.")
	if len(model.asked) != 0 {
		t.Errorf("nothing shared before: nothing asked, %q", model.asked)
	}
	p, _ := a.Store.Create(records.ProposalType, map[string]any{"summary": "Make a task?", "action": map[string]any{"tool": "create_record", "type": "task", "fields": map[string]any{"title": "Pay rent"}}, "state": "pending", "from": "note/" + first.ID})
	second := share("Message", "Lease renewal part 2: sign and return by 28 November.")
	if _, err := a.Store.Get("note", first.ID); err == nil {
		t.Errorf("the first part is joined into the second, not kept apart")
	}
	body := fmt.Sprint(second.Fields["body"])
	if i, j := strings.Index(body, "part 1"), strings.Index(body, "part 2"); i < 0 || j < i || second.Fields["title"] != "Lease renewal, 14 Mill Lane" {
		t.Errorf("one note, the first part first, named for the whole: %v %q", second.Fields["title"], body)
	}
	if got, _ := a.Store.Get(records.ProposalType, p.ID); got == nil || got.Fields["state"] != "dismissed" {
		t.Errorf("what was suggested from the first part alone is put away")
	}
	gym := share("Gym", "Gym timetable: spin on Tuesdays at 7.")
	if _, err := a.Store.Get("note", second.ID); err != nil || strings.Contains(fmt.Sprint(gym.Fields["body"]), "Lease") {
		t.Errorf("something else shared after stays apart")
	}
}
