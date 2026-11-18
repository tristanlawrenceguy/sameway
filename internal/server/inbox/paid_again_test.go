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

// Sorting set up before it knew of them gains, once, tags for what is paid
// again and again and what is bought, and actions suggesting a record of
// each from email and from what is shared.
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
	if paid != 1 || !hasTag(a.Store, "bought") {
		t.Errorf("the tag is added once, got %d", paid)
	}
	actions, _ := a.Store.List(records.ActionType, store.ListOptions{})
	suggest := 0
	for _, act := range actions {
		if act.Fields["kind"] == "classify" && act.Fields["what"] == "email" && !strings.Contains(strings.Join(records.StringList(act.Fields["tags"]), ","), "paid again") {
			t.Errorf("sorting email gives the tag: %v", act.Fields["tags"])
		}
		if act.Fields["make"] == "subscription" || act.Fields["make"] == "purchase" {
			suggest++
			if act.Fields["update"] != true {
				t.Errorf("it changes one kept: %v", act.Fields)
			}
		}
		if act.Fields["make"] == "task" && act.Fields["update"] != true {
			t.Errorf("a task suggested changes the one open: %v", act.Fields)
		}
	}
	if suggest != 4 {
		t.Errorf("a subscription and a purchase are each suggested from email and from what is shared, got %d", suggest)
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
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Suggest a subscription", "kind": "suggest", "make": "subscription", "what": "email"})
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

func hasTag(a interface {
	List(string, store.ListOptions) ([]*store.Record, error)
}, name string) bool {
	tags, _ := a.List(records.TagType, store.ListOptions{})
	for _, tg := range tags {
		if tg.Fields["name"] == name {
			return true
		}
	}
	return false
}

// Sorting set up before tasks changed one open has its task suggestions
// change one, once; what it gave as subscriptions before changes one kept
// and is shared too.
func TestSortingSetUpBeforeIsBroughtOn(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	srv.SetUpSorting()
	actions, _ := a.Store.List(records.ActionType, store.ListOptions{})
	for _, act := range actions {
		if act.Fields["make"] == "task" {
			a.Store.Update(records.ActionType, act.ID, map[string]any{"update": false})
		}
	}
	a.Store.SetMeta("sorting:paid-again", "yes")
	srv.SortPaidAgain()
	actions, _ = a.Store.List(records.ActionType, store.ListOptions{})
	tasks, shared := 0, 0
	for _, act := range actions {
		if act.Fields["make"] == "task" && act.Fields["update"] == true {
			tasks++
		}
		if act.Fields["make"] == "subscription" && act.Fields["what"] == "note" {
			shared++
		}
	}
	if tasks != 2 || shared != 1 {
		t.Errorf("both task suggestions change one open (%d), and subscriptions come from what is shared (%d)", tasks, shared)
	}
}

// askedModel keeps what it was asked and suggests nothing.
type askedModel struct{ asked string }

func (m *askedModel) Name() string { return "asked" }
func (m *askedModel) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.asked = req.Messages[0].Content
	return &llm.Response{Text: `{"make": false}`}, nil
}

// A bill's reminder is matched against the tasks still open: one paid
// and done is not opened again.
func TestABillReminderIsMatchedToTheOpenTask(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	m := &askedModel{}
	a.Chat.Provider, a.Chat.ProviderErr = m, nil
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Suggest a task", "kind": "suggest", "make": "task", "update": true})
	open, _ := a.Store.Create("task", map[string]any{"title": "Pay the electricity bill"})
	paid, _ := a.Store.Create("task", map[string]any{"title": "Pay the October water bill", "done": true})
	mail, _ := a.Store.Create("email", map[string]any{"subject": "Reminder: your bill is due", "body": "Your electricity bill of $84 is due on the 20th."})
	a.Automate.Suggest(context.Background(), act, "email", mail.ID)
	if !strings.Contains(m.asked, open.ID) || strings.Contains(m.asked, paid.ID) {
		t.Errorf("the open task is shown to change, the done one not: %s", m.asked)
	}
}

// A subscription suggested from an email is whose mailbox the email came to.
func TestASubscriptionIsWhoseMailboxItCameTo(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &keptModel{}, nil
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Suggest a subscription", "kind": "suggest", "make": "subscription"})
	mail, _ := a.Store.Create("email", map[string]any{"subject": "Welcome to Streamflix", "body": "$15.99 a month.", "mailbox": "sam@example.com"})
	r := a.Automate.Suggest(context.Background(), act, "email", mail.ID)
	p, err := a.Store.Get(records.ProposalType, r.Answer)
	if err != nil {
		t.Fatalf("suggested: %v", r.Text)
	}
	fields := p.Fields["action"].(map[string]any)["fields"].(map[string]any)
	if fields["mailbox"] != "sam@example.com" {
		t.Errorf("the mailbox is carried: %v", fields)
	}
}

// fixedModel answers every question with the same words.
type fixedModel struct{ said string }

func (m fixedModel) Name() string { return "fixed" }
func (m fixedModel) Complete(context.Context, llm.Request) (*llm.Response, error) {
	return &llm.Response{Text: m.said}, nil
}

// A receipt's day a little gone by stays in the past: bought yesterday is
// not bought next year. A day long gone is the coming one, as before.
func TestAReceiptsDayGoneByStaysPast(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = fixedModel{`{"make": true, "fields": {"item": "Headphones", "bought": "17 November", "return_by": "15 January"}}`}, nil
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Suggest a purchase", "kind": "suggest", "make": "purchase"})
	mail, _ := a.Store.Create("email", map[string]any{"subject": "Your order", "body": "Headphones, bought 17 November. Return by 15 January.", "received": "2026-11-18T09:00:00Z"})
	r := a.Automate.Suggest(context.Background(), act, "email", mail.ID)
	p, err := a.Store.Get(records.ProposalType, r.Answer)
	if err != nil {
		t.Fatalf("suggested: %v", r.Text)
	}
	fields := p.Fields["action"].(map[string]any)["fields"].(map[string]any)
	if b, _ := fields["bought"].(string); !strings.HasPrefix(b, "2026-11-17") {
		t.Errorf("bought the day before it was sent: %v", fields["bought"])
	}
	if d, _ := fields["return_by"].(string); !strings.HasPrefix(d, "2027-01-15") {
		t.Errorf("returned by the coming 15 January: %v", fields["return_by"])
	}
}
