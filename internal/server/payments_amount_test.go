package server_test

import (
	"strings"
	"testing"
	"time"
)

// A payment's page says its amount with its currency, as one sum, as its
// question and its notice say it: a person hears 60.00 GBP, never a bare
// 60 with the currency left out because it is the default.
func TestAPaymentSaysItsSum(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := time.Now()
	for id, f := range map[string]map[string]any{
		"pay1": {"what": "Deposit", "amount": 60.0, "currency": "gbp"},
		"pay2": {"what": "Final bill", "amount": 1240.5, "currency": "eur"},
	} {
		if _, err := a.Store.Put("payment", id, f, now, now); err != nil {
			t.Fatal(err)
		}
	}
	for id, want := range map[string]string{"pay1": ">60.00 GBP<", "pay2": ">1240.50 EUR<"} {
		page := get(t, h, "/t/payment/"+id).Body.String()
		if !strings.Contains(page, want) || strings.Contains(page, ">Currency<") {
			t.Errorf("%s says %s, once: %s", id, want, truncate(page))
		}
	}
}
