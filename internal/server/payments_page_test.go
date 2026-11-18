package server_test

import (
	"strings"
	"testing"
	"time"
)

// A payment is kept by Sameway, so its page and its list offer nothing a
// person could press only to be refused: no Edit, no Delete, no Add, no
// Import.
func TestAPaymentOffersOnlyWhatWorks(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := time.Now()
	if _, err := a.Store.Put("payment", "pay1", map[string]any{"what": "Deposit", "amount": 60.0, "currency": "gbp", "link": "https://buy.stripe.com/test_1"}, now, now); err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/payment/pay1").Body.String()
	for _, bad := range []string{"/t/payment/pay1/delete", "data-edit-fields", "data-edit-action"} {
		if strings.Contains(page, bad) {
			t.Errorf("its page offers %s: %s", bad, truncate(page))
		}
	}
	list := get(t, h, "/t/payment").Body.String()
	for _, bad := range []string{"/t/payment/add", "/t/payment/import"} {
		if strings.Contains(list, bad) {
			t.Errorf("its list offers %s: %s", bad, truncate(list))
		}
	}
}
