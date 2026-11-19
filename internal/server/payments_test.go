package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// fakeStripe is Stripe's API for a test: one key it takes, a link made,
// and whether it is paid.
type fakeStripe struct {
	mu    sync.Mutex
	paid  int64 // the amount paid, in cents; 0 for none
	made  url.Values
	links url.Values
}

func (f *fakeStripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if key, _, _ := r.BasicAuth(); key != "rk_test_ok" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "Invalid API Key"}})
		return
	}
	r.ParseForm()
	switch r.URL.Path {
	case "/v1/balance":
		json.NewEncoder(w).Encode(map[string]any{"object": "balance"})
	case "/v1/prices":
		f.made = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{"id": "price_1"})
	case "/v1/payment_links":
		f.links = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{"id": "plink_1", "url": "https://buy.stripe.com/test_1"})
	case "/v1/checkout/sessions":
		status := "unpaid"
		if f.paid > 0 && r.URL.Query().Get("payment_link") == "plink_1" {
			status = "paid"
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"payment_status": status, "amount_total": f.paid, "currency": "gbp"}}})
	default:
		http.NotFound(w, r)
	}
}

// Being paid, made safe against attacks and against a person not paying
// attention: only a restricted Stripe key is taken; a link is made only
// with the owner's press, whose button says the amount, and never above
// their most; it takes one payment; payments are Sameway's alone to write,
// so nothing a hand or a model writes can make one, or mark one paid;
// paid is what Stripe says, in full; an email shows what each link to
// pay in it really asks for, and one with a link made elsewhere is refused.
func TestALinkToPayTheOwner(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	stripe := &fakeStripe{}
	a.Automate.HTTP = &http.Client{Transport: served{stripe}}
	rang := make(chan string, 2)
	h.(*server.Server).OnRing(func(title, text, _ string) { rang <- title + ": " + text })
	if page := get(t, h, "/payments").Body.String(); !strings.Contains(page, "restricted key") {
		t.Fatalf("the page says how: %s", truncate(page))
	}
	if page := after(t, h, postForm(t, h, "/payments/connect", url.Values{"key": {"sk_live_secret"}})).Body.String(); !strings.Contains(page, "restricted key") {
		t.Errorf("a secret key, which could pay out, is not taken: %s", truncate(page))
	}
	if page := after(t, h, postForm(t, h, "/payments/connect", url.Values{"key": {"rk_wrong"}})).Body.String(); !strings.Contains(page, "did not take that key") {
		t.Errorf("a wrong key says so: %s", truncate(page))
	}
	postForm(t, h, "/payments/connect", url.Values{"key": {"rk_test_ok"}})

	if text, isErr := a.Chat.Call("payment_link", json.RawMessage(`{"what":"Boiler service","amount":6000}`)); !isErr || !strings.Contains(text, "above the most") || len(a.Chat.Proposals()) != 0 {
		t.Fatalf("a slip of a zero is refused, not asked: %s", text)
	}
	a.Chat.Call("payment_link", json.RawMessage(`{"what":"Boiler service","amount":60}`))
	ps := a.Chat.Proposals()
	if len(ps) != 1 || ps[0].Fields["summary"] != `Make a link to pay you 60.00 GBP for "Boiler service"?` || ps[0].Fields["yes"] != "Make a 60.00 GBP link" || stripe.made != nil {
		t.Fatalf("nothing is made before the owner's press, whose button says the amount: %+v", ps)
	}
	wantStatus(t, postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	if stripe.made.Get("unit_amount") != "6000" || stripe.made.Get("currency") != "gbp" || stripe.links.Get("restrictions[completed_sessions][limit]") != "1" {
		t.Errorf("Stripe is asked for one payment of it: %v %v", stripe.made, stripe.links)
	}
	pays, _ := a.Store.List("payment", store.ListOptions{})
	if len(pays) != 1 || pays[0].Fields["link"] != "https://buy.stripe.com/test_1" {
		t.Fatalf("it is kept as a payment: %+v", pays)
	}
	for _, call := range []string{`{"type":"payment","fields":{"what":"Fake","amount":1,"paid":true}}`} {
		if text, isErr := a.Chat.Call("create_record", json.RawMessage(call)); !isErr || !strings.Contains(text, "kept by Sameway") {
			t.Errorf("no hand or model makes a payment: %s", text)
		}
	}
	if text, isErr := a.Chat.Call("update_record", json.RawMessage(`{"type":"payment","id":"`+pays[0].ID+`","fields":{"paid":true}}`)); !isErr || !strings.Contains(text, "kept by Sameway") {
		t.Errorf("or marks one paid: %s", text)
	}

	a.Chat.MailFrom = func() string { return "tris+sameway@example.com" }
	if text, _ := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"Bill","text":"Just £6 please: https://buy.stripe.com/test_1"}`)); !strings.Contains(text, `It carries a link to pay 60.00 GBP for "Boiler service" (not paid yet), whatever the words say`) {
		t.Errorf("an email shows what its link really asks for: %s", text)
	}
	if text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"Bill","text":"Pay here: https://buy.stripe.com/someone_elses"}`)); !isErr || !strings.Contains(text, "not made here") {
		t.Errorf("and one with a link made elsewhere is refused: %s", text)
	}
	for _, forged := range []string{`"subject":"Pay at https://buy.stripe.com/someone_elses","text":"Thanks"`, `"subject":"Bill","text":"Pay at HTTPS://BUY.STRIPE.COM/someone_elses"`} {
		if text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com",`+forged+`}`)); !isErr || !strings.Contains(text, "not made here") {
			t.Errorf("in the subject or in capitals too: %s", text)
		}
	}
	if text, _ := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"Bill","text":"Pay at https://buy.stripe.com/test_1."}`)); !strings.Contains(text, "It carries a link to pay 60.00 GBP") {
		t.Errorf("a full stop after the link is not part of it: %s", text)
	}

	stripe.mu.Lock()
	stripe.paid = 5000
	stripe.mu.Unlock()
	a.Chat.FollowPayments(context.Background())
	if p, _ := a.Store.Get("payment", pays[0].ID); p.Fields["paid"] == true || len(rang) != 0 {
		t.Fatal("less than asked is not paid")
	}
	stripe.mu.Lock()
	stripe.paid = 6000
	stripe.mu.Unlock()
	a.Chat.FollowPayments(context.Background())
	if p, _ := a.Store.Get("payment", pays[0].ID); p.Fields["paid"] != true {
		t.Fatal("paid in full, as Stripe says")
	}
	if got := <-rang; !strings.Contains(got, `Paid: 60.00 GBP for "Boiler service"`) {
		t.Errorf("and the owner is told: %s", got)
	}
}

// Asking someone to pay by email was two yeses and a link copied between
// them; now it is one: the email's own press makes each person's link,
// with the amount on its button, and sends it. Above the most is refused.
func TestAnEmailAsksToBePaid(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	stripe := &fakeStripe{}
	a.Automate.HTTP = &http.Client{Transport: served{stripe}}
	postForm(t, h, "/payments/connect", url.Values{"key": {"rk_test_ok"}})
	var sent []chat.Outgoing
	a.Chat.MailFrom = func() string { return "tris+sameway@example.com" }
	a.Chat.Mail = func(m chat.Outgoing) (string, error) { sent = append(sent, m); return "<1@example.com>", nil }

	if text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"Bill","text":"Thanks for having us.","pay":{"what":"Boiler service","amount":6000}}`)); !isErr || !strings.Contains(text, "above the most") || len(a.Chat.Proposals()) != 0 {
		t.Fatalf("a slip of a zero is refused, not asked: %s", text)
	}
	a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"Bill","text":"Thanks for having us.","pay":{"what":"Boiler service","amount":60}}`))
	ps := a.Chat.Proposals()
	if len(ps) != 1 || ps[0].Fields["yes"] != "Send it, asking 60.00 GBP" || stripe.made != nil {
		t.Fatalf("nothing is made before the press, whose button says the amount: %+v", ps)
	}
	wantStatus(t, postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "To pay 60.00 GBP for Boiler service: https://buy.stripe.com/test_1") {
		t.Fatalf("the email carries the link it made: %+v", sent)
	}
	pays, _ := a.Store.List("payment", store.ListOptions{})
	if len(pays) != 1 || pays[0].Fields["asked_of"] != "pat@example.com" || stripe.links.Get("restrictions[completed_sessions][limit]") != "1" {
		t.Errorf("kept as a payment asked of them, taking one payment: %+v", pays)
	}
}
