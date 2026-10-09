package server_test

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// tagModel answers a classify action's questions: the tag it suggests, and
// what it decides when shown the person's past choices.
type tagModel struct {
	mu      sync.Mutex
	decide  string // keep, confirm, change
	checked []string
}

func (m *tagModel) Name() string { return "tags" }

func (m *tagModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := req.Messages[0].Content
	if strings.Contains(req.System, "against what the person did") {
		m.checked = append(m.checked, q)
		return &llm.Response{Text: `{"decide": "` + m.decide + `", "to": "important", "why": "as you chose before"}`}, nil
	}
	return &llm.Response{Text: `{"tags": [{"tag": "to do", "why": "asks for a payment"}]}`}, nil
}

func tagWait(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if ok() {
			return
		}
	}
	t.Fatalf("waited for %s", what)
}

// An email coming in sets off the sorting action like any record would; the
// tag it gives waits on Today, and what the person does with it (keeps it,
// takes it off, changes it) is shown to the action next time, which then
// follows them: changing a tag as they did, or keeping it for them.
func TestTagsFollowThePersonsJudgement(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	model := &tagModel{decide: "keep"}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	srv := server.New(a)
	a.Chat.StartAutomating()
	srv.SetUpSorting()

	classOf := func(id string) *store.Record {
		cls, _ := a.Store.List(chat.ClassificationType, store.ListOptions{})
		for _, c := range cls {
			if c.Fields["record"] == "email/"+id {
				return c
			}
		}
		return nil
	}
	arrive := func(subject string) *store.Record {
		rec, err := a.Store.Create("email", map[string]any{"subject": subject, "from": "Water Co", "body": "Please pay by Friday."})
		if err != nil {
			t.Fatal(err)
		}
		tagWait(t, "the tag for "+subject, func() bool { return classOf(rec.ID) != nil })
		return rec
	}

	first := arrive("Water bill")
	if c := classOf(first.ID); c.Fields["tag"] != "to do" || c.Fields["state"] != "suggested" {
		t.Fatalf("suggested: %v", c.Fields)
	}
	today := get(t, srv, "/today").Body.String()
	if !strings.Contains(today, "Tags to check") || !strings.Contains(today, "asks for a payment") || !strings.Contains(today, ">Take it off<") {
		t.Fatalf("Today asks: %s", truncate(today))
	}
	postForm(t, srv, "/tags/off", url.Values{"id": {classOf(first.ID).ID}})
	tagWait(t, "taken off", func() bool { return classOf(first.ID).Fields["state"] == "removed" })

	second := arrive("Gas bill")
	postForm(t, srv, "/tags/change", url.Values{"id": {classOf(second.ID).ID}, "to": {"important"}})
	tagWait(t, "changed", func() bool {
		c := classOf(second.ID)
		return c.Fields["state"] == "changed" && c.Fields["to"] == "important"
	})
	if got, _ := a.Store.Get("email", second.ID); !strings.Contains(strings.Join(anyStrings(got.Fields["tags"]), ","), "important") {
		t.Errorf("the record carries the tag it was changed to: %v", got.Fields["tags"])
	}

	model.mu.Lock()
	model.decide = "change"
	model.mu.Unlock()
	third := arrive("Phone bill")
	model.mu.Lock()
	last := model.checked[len(model.checked)-1]
	model.mu.Unlock()
	if !strings.Contains(last, "Gas bill") || !strings.Contains(last, "changed it to important") || !strings.Contains(last, "took the tag off") {
		t.Errorf("the check is shown the person's choices, newest first: %s", last)
	}
	if strings.Index(last, "Gas bill") > strings.Index(last, "Water bill") {
		t.Error("newest first")
	}
	if c := classOf(third.ID); c.Fields["tag"] != "important" {
		t.Errorf("it follows the person, changing the tag as they did: %v", c.Fields)
	}

	model.mu.Lock()
	model.decide = "confirm"
	model.mu.Unlock()
	fourth := arrive("Council tax")
	if c := classOf(fourth.ID); c.Fields["state"] != "confirmed" || c.Fields["confirmed_by"] != "judgement" {
		t.Errorf("kept for the person when their choices are plain: %v", c.Fields)
	}
	if strings.Contains(get(t, srv, "/today").Body.String(), "Council tax") {
		t.Error("a tag kept from their choices is not asked again")
	}
}

func anyStrings(v any) []string {
	var out []string
	if l, ok := v.([]any); ok {
		for _, x := range l {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// fitsModel says every tag it is asked about on its own fits.
type fitsModel struct{ asked []string }

func (m *fitsModel) Name() string { return "fits" }
func (m *fitsModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.asked = append(m.asked, req.Messages[0].Content)
	return &llm.Response{Text: `{"fits": true, "why": "it says so"}`}, nil
}

// An action may give only some of the tags, and ask about each on its own.
func TestAClassifyActionGivesItsOwnTagsEachApart(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	m := &fitsModel{}
	a.Chat.Provider, a.Chat.ProviderErr = m, nil
	srv := server.New(a)
	srv.SetUpSorting()
	a.Store.Create(chat.TagType, map[string]any{"name": "waiting", "means": "Someone owes me an answer."})
	act, _ := a.Store.Create("action", map[string]any{"title": "Mark the waiting and important", "kind": "classify", "tags": []any{"waiting", "important"}, "apart": true})
	note, _ := a.Store.Create("note", map[string]any{"title": "Asked Joe for the quote"})
	_ = a.Chat.Classify(context.Background(), act, "note", note.ID)
	got, _ := a.Store.Get("note", note.ID)
	if tags := strings.Join(anyStrings(got.Fields["tags"]), ","); !strings.Contains(tags, "waiting") || !strings.Contains(tags, "important") || strings.Contains(tags, "to do") {
		t.Errorf("its own tags given: %s", tags)
	}
	if len(m.asked) != 2 || strings.Contains(strings.Join(m.asked, "|"), `"to do"`) {
		t.Errorf("each of its own tags asked about apart, no other: %v", m.asked)
	}
}
