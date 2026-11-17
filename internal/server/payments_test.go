package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// fakeStripe is Stripe's API for a test: one key it takes, a link made,
// and whether it is paid.
type fakeStripe struct {
	mu   sync.Mutex
	paid bool
	made url.Values
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
		json.NewEncoder(w).Encode(map[string]any{"id": "plink_1", "url": "https://buy.stripe.com/test_1"})
	case "/v1/checkout/sessions":
		status := "unpaid"
		if f.paid && r.URL.Query().Get("payment_link") == "plink_1" {
			status = "paid"
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"payment_status": status}}})
	default:
		http.NotFound(w, r)
	}
}

// Being paid: the owner connects their own Stripe with a key; a link to
// pay them is made only with their yes, kept as a payment; Sameway marks
// it paid when Stripe says so, and tells the owner.
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
	if page := after(t, h, postForm(t, h, "/payments/connect", url.Values{"key": {"sk_wrong"}})).Body.String(); !strings.Contains(page, "did not take that key") {
		t.Errorf("a wrong key says so: %s", truncate(page))
	}
	postForm(t, h, "/payments/connect", url.Values{"key": {"rk_test_ok"}})

	text, _ := a.Chat.Call("payment_link", json.RawMessage(`{"what":"Boiler service","amount":60}`))
	ps := a.Chat.Proposals()
	if len(ps) != 1 || ps[0].Fields["summary"] != `Make a link to pay you 60.00 GBP for "Boiler service"?` || stripe.made != nil {
		t.Fatalf("nothing is made before the owner's yes: %s", text)
	}
	wantStatus(t, postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	if stripe.made.Get("unit_amount") != "6000" || stripe.made.Get("currency") != "gbp" || stripe.made.Get("product_data[name]") != "Boiler service" {
		t.Errorf("Stripe is asked for it: %v", stripe.made)
	}
	pays, _ := a.Store.List("payment", store.ListOptions{})
	if len(pays) != 1 || pays[0].Fields["link"] != "https://buy.stripe.com/test_1" || pays[0].Fields["paid"] != false {
		t.Fatalf("it is kept as a payment: %+v", pays)
	}

	a.Chat.FollowPayments(context.Background())
	if len(rang) != 0 {
		t.Fatal("nothing is told before it is paid")
	}
	stripe.mu.Lock()
	stripe.paid = true
	stripe.mu.Unlock()
	a.Chat.FollowPayments(context.Background())
	if p, _ := a.Store.Get("payment", pays[0].ID); p.Fields["paid"] != true {
		t.Fatal("paid, as Stripe says")
	}
	if got := <-rang; !strings.Contains(got, `Paid: 60.00 GBP for "Boiler service"`) {
		t.Errorf("and the owner is told: %s", got)
	}
}
