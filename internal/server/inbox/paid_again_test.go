package inbox_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Sorting set up before it knew of them gains, once, a tag for what is paid
// again and again and an action suggesting a subscription from each.
func TestSortingGainsPaidAgainOnce(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	srv.SortPaidAgain()
	if tags, _ := a.Store.List(records.TagType, store.ListOptions{}); len(tags) != 0 {
		t.Fatal("without sorting set up, nothing is added")
	}
	srv.SetUpSorting()
	srv.SortPaidAgain()
	srv.SortPaidAgain()
	tags, _ := a.Store.List(records.TagType, store.ListOptions{})
	paid := 0
	for _, tg := range tags {
		if tg.Fields["name"] == "paid again" {
			paid++
		}
	}
	if paid != 1 {
		t.Errorf("the tag is added once, got %d", paid)
	}
	actions, _ := a.Store.List(records.ActionType, store.ListOptions{})
	suggest := 0
	for _, act := range actions {
		if act.Fields["kind"] == "classify" && act.Fields["what"] == "email" && !strings.Contains(strings.Join(records.StringList(act.Fields["tags"]), ","), "paid again") {
			t.Errorf("sorting email gives the tag: %v", act.Fields["tags"])
		}
		if act.Fields["make"] == "subscription" {
			suggest++
		}
	}
	if suggest != 1 {
		t.Errorf("one action suggests a subscription, got %d", suggest)
	}
}

// keptModel suggests a subscription, or a change to the one kept when the
// question lists it.
type keptModel struct{ price string }

func (m *keptModel) Name() string { return "kept" }

func (m *keptModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	q := req.Messages[0].Content
	if _, rest, ok := strings.Cut(q, "Already kept:\n- "); ok {
		id, _, _ := strings.Cut(rest, ":")
		return &llm.Response{Text: `{"make": true, "same": "` + id + `", "fields": {"name": "Streamflix", "price": "` + m.price + `"}, "why": "a new price"}`}, nil
	}
	return &llm.Response{Text: `{"make": true, "fields": {"name": "Streamflix", "price": "$15.99"}, "why": "a monthly plan"}`}, nil
}

// A second email from a service kept changes the one kept with what is
// new, put to the person; one saying nothing new suggests nothing.
func TestASecondReceiptChangesTheSubscriptionKept(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	m := &keptModel{price: "$17.99"}
	a.Chat.Provider, a.Chat.ProviderErr = m, nil
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Suggest a subscription", "kind": "suggest", "make": "subscription", "update": true})
	kept, _ := a.Store.Create("subscription", map[string]any{"name": "Streamflix", "price": "$15.99"})
	mail, _ := a.Store.Create("email", map[string]any{"subject": "Your price is changing", "body": "From next month Streamflix costs $17.99."})
	r := a.Automate.Suggest(context.Background(), act, "email", mail.ID)
	p, err := a.Store.Get(records.ProposalType, r.Answer)
	if err != nil {
		t.Fatalf("a change is suggested: %v", r.Text)
	}
	if s := p.Fields["summary"].(string); !strings.Contains(s, "Change Streamflix") || !strings.Contains(s, "price $17.99") || strings.Contains(s, "name") {
		t.Errorf("it says what changes, and only that: %s", s)
	}
	if err := a.Chat.Accept(p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.Store.Get("subscription", kept.ID); got.Fields["price"] != "$17.99" {
		t.Errorf("the one kept is changed: %v", got.Fields)
	}
	if subs, _ := a.Store.List("subscription", store.ListOptions{}); len(subs) != 1 {
		t.Errorf("no second one is made, got %d", len(subs))
	}
	again, _ := a.Store.Create("email", map[string]any{"subject": "Your receipt", "body": "Streamflix $17.99 paid."})
	if r := a.Automate.Suggest(context.Background(), act, "email", again.ID); r.Answer != "" || !strings.Contains(r.Text, "nothing new") {
		t.Errorf("nothing new suggests nothing: %+v", r)
	}
}

// Sorting's first words for to do and paid again are brought up to date;
// words the person wrote are theirs.
func TestSortingWordsAreBroughtUpToDate(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	a.Store.SetMeta("sorting:paid-again", "yes")
	old, _ := a.Store.Create(records.TagType, map[string]any{"name": "paid again", "means": "A receipt, renewal notice or free-trial email for something paid again and again: a subscription, membership or plan. Not a one-off purchase."})
	mine, _ := a.Store.Create(records.TagType, map[string]any{"name": "to do", "means": "What I must do."})
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Suggest a subscription", "kind": "suggest", "make": "subscription"})
	srv.SortPaidAgain()
	if got, _ := a.Store.Get(records.TagType, old.ID); !strings.Contains(got.Fields["means"].(string), "cancellation") {
		t.Errorf("the first words are brought up to date: %v", got.Fields["means"])
	}
	if got, _ := a.Store.Get(records.TagType, mine.ID); got.Fields["means"] != "What I must do." {
		t.Errorf("the person's words are kept: %v", got.Fields["means"])
	}
	if got, _ := a.Store.Get(records.ActionType, act.ID); got.Fields["update"] != true {
		t.Errorf("the suggestion changes one kept: %v", got.Fields)
	}
	a.Store.Update(records.ActionType, act.ID, map[string]any{"update": false})
	srv.SortPaidAgain()
	if got, _ := a.Store.Get(records.ActionType, act.ID); got.Fields["update"] != false {
		t.Errorf("turned off after, it stays off: %v", got.Fields)
	}
}
