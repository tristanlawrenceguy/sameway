// Package stripe asks Stripe, with the owner's own key, for a link to pay
// them, and whether it was paid. It is Stripe's API over https, and
// nothing else: no card or account details ever pass through Sameway.
package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// API is where Stripe answers.
const API = "https://api.stripe.com"

// Account is the owner's Stripe, by their key.
type Account struct {
	Key  string
	HTTP *http.Client // nil is one with a minute's limit
	Base string       // "" is API; a test gives its own
}

func (a Account) call(ctx context.Context, method, path string, form url.Values, out any) error {
	base, client := a.Base, a.HTTP
	if base == "" {
		base = API
	}
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.Key, "")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := client.Do(req)
	if err != nil {
		return errors.New("Stripe could not be reached")
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(raw, &e)
		if res.StatusCode == http.StatusUnauthorized {
			return errors.New("Stripe did not take that key: copy it again from the Stripe dashboard")
		}
		return fmt.Errorf("Stripe said: %s", e.Error.Message)
	}
	return json.Unmarshal(raw, out)
}

// Check says whether the key works.
func (a Account) Check(ctx context.Context) error {
	var out map[string]any
	return a.call(ctx, http.MethodGet, "/v1/balance", nil, &out)
}

// Link makes a link to pay amount (in the currency's whole units) for
// what, and gives Stripe's name for it and its address.
func (a Account) Link(ctx context.Context, what string, amount float64, currency string) (id, link string, err error) {
	var price struct {
		ID string `json:"id"`
	}
	err = a.call(ctx, http.MethodPost, "/v1/prices", url.Values{"unit_amount": {fmt.Sprint(int64(amount*100 + 0.5))}, "currency": {strings.ToLower(currency)}, "product_data[name]": {what}}, &price)
	if err != nil {
		return "", "", err
	}
	var pl struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	// One payment and the link closes: nobody pays twice by the same link.
	err = a.call(ctx, http.MethodPost, "/v1/payment_links", url.Values{"line_items[0][price]": {price.ID}, "line_items[0][quantity]": {"1"}, "restrictions[completed_sessions][limit]": {"1"}}, &pl)
	return pl.ID, pl.URL, err
}

// Paid says whether a link has been paid in full: the amount (in whole
// units) and currency asked, and nothing less.
func (a Account) Paid(ctx context.Context, link string, amount float64, currency string) (bool, error) {
	var out struct {
		Data []struct {
			PaymentStatus string `json:"payment_status"`
			AmountTotal   int64  `json:"amount_total"`
			Currency      string `json:"currency"`
		} `json:"data"`
	}
	if err := a.call(ctx, http.MethodGet, "/v1/checkout/sessions?"+url.Values{"payment_link": {link}, "limit": {"10"}}.Encode(), nil, &out); err != nil {
		return false, err
	}
	for _, s := range out.Data {
		if s.PaymentStatus == "paid" && s.AmountTotal == int64(amount*100+0.5) && strings.EqualFold(s.Currency, currency) {
			return true, nil
		}
	}
	return false, nil
}

// Restricted says whether a key is a restricted one, which may do only
// what it was given: a secret key may also refund and pay out, which
// Sameway never needs, so it is not taken.
func Restricted(key string) bool { return strings.HasPrefix(key, "rk_") }
