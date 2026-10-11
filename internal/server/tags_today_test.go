package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// classOn is the classification of a tag on a record, waited for.
func classOn(t *testing.T, st *store.Store, ref, tag string) *store.Record {
	t.Helper()
	var found *store.Record
	tagWait(t, tag+" on "+ref, func() bool {
		cls, _ := st.List(chat.ClassificationType, store.ListOptions{})
		for _, c := range cls {
			if c.Fields["record"] == ref && c.Fields["tag"] == tag {
				found = c
			}
		}
		return found != nil
	})
	return found
}

// A tag on an email waiting to be sorted is checked beside that email,
// said once: not again under Tags to check. Making the task suggested
// from it keeps the tag that set the suggestion off, as the person's own
// choice, so it is not asked about again.
func TestATagIsCheckedBesideItsEmailAndAgreedByMakingTheTask(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &sortModel{}, nil
	srv := server.New(a)
	a.Chat.StartAutomating()
	srv.SetUpSorting()

	rec, _ := a.Store.Create("email", map[string]any{"subject": "Water bill", "from": "Water Co", "body": "Please pay by Friday.", "tags": []any{"to sort"}})
	c := classOn(t, a.Store, "email/"+rec.ID, "to do")
	var from string
	tagWait(t, "the task suggested", func() bool {
		for _, p := range a.Chat.Suggestions() {
			if p.Fields["from"] == "email/"+rec.ID {
				from = p.ID
			}
		}
		return from != ""
	})
	page := said(get(t, srv, "/today").Body.String())
	if strings.Count(page, "Tagged to do: asks for a payment.") != 1 || strings.Contains(page, "Tags to check") {
		t.Fatalf("the tag is checked once, beside its email: %s", page)
	}
	if i, j := strings.Index(page, "Tagged to do"), strings.Index(page, "Make it"); i < strings.Index(page, "To sort") || j < i {
		t.Errorf("in the email's row, before what was suggested from it: %s", page)
	}

	postForm(t, srv, "/suggested/yes", url.Values{"id": {from}})
	got, _ := a.Store.Get(chat.ClassificationType, c.ID)
	if got.Fields["state"] != "confirmed" || got.Fields["confirmed_by"] != "person" {
		t.Errorf("making the task keeps the tag that set it off, as the person's: %v", got.Fields)
	}
	if page := said(get(t, srv, "/today").Body.String()); strings.Contains(page, "Water bill") {
		t.Errorf("nothing left to ask about it: %s", page)
	}
}

// A tag kept for the person from their past choices is not asked about,
// but it is not silent either: Today folds it under Kept for you from
// your choices, counted, each with Take it off and Change, never Keep.
// Taking one off is their choice, which the action follows next time.
func TestATagKeptFromYourChoicesIsShownFoldedAndCanBeTakenOff(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	a.Chat.StartAutomating() // what notices a tag taken off
	srv.SetUpSorting()
	rec, _ := a.Store.Create("email", map[string]any{"subject": "Council tax", "body": "Your bill is due.", "tags": []any{"to do"}})
	c, err := a.Store.Create(chat.ClassificationType, map[string]any{"record": "email/" + rec.ID, "tag": "to do", "why": "as you kept the gas bill", "state": "confirmed", "confirmed_by": "judgement"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, srv, "/today").Body.String()
	page := said(body)
	if !strings.Contains(page, "Tags to check") || !strings.Contains(page, "Kept for you from your choices 1 tag") || !strings.Contains(page, "Kept to do: as you kept the gas bill.") {
		t.Fatalf("folded, counted, and said with why: %s", page)
	}
	if strings.Contains(page, "Keep to do on Council tax") || !strings.Contains(page, "Take it off to do on Council tax") {
		t.Errorf("taken off or changed, not kept again: %s", page)
	}
	if !strings.Contains(body, `<details class="sw-disclosure"`) {
		t.Error("folded in a disclosure")
	}
	back, _ := landed(t, srv, postForm(t, srv, "/tags/off", url.Values{"id": {c.ID}}))
	if !strings.Contains(back.Body.String(), "the action follows that next time") {
		t.Errorf("says what taking it off does: %s", truncate(back.Body.String()))
	}
	tagWait(t, "taken off", func() bool {
		got, _ := a.Store.Get(chat.ClassificationType, c.ID)
		return got.Fields["state"] == "removed"
	})
	if page := said(get(t, srv, "/today").Body.String()); strings.Contains(page, "Kept for you") {
		t.Errorf("gone once dealt with: %s", page)
	}
}
