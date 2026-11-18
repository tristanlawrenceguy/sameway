package stripe

import "testing"

// Stripe counts each currency in its own smallest unit: a link for 300
// yen asks for 300, not 30000, and one for 60 dinars asks for 60000.
func TestMinorIsWhatStripeTakes(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		amount   float64
		currency string
		want     int64
	}{
		{60, "gbp", 6000},
		{60.5, "EUR", 6050},
		{0.29, "usd", 29},
		{300, "jpy", 300},
		{300.4, "krw", 300},
		{5, "isk", 500},
		{5.3, "ugx", 500},
		{60, "kwd", 60000},
		{1.234, "bhd", 1230},
	} {
		if got := Minor(c.amount, c.currency); got != c.want {
			t.Errorf("%v %s: %d, want %d", c.amount, c.currency, got, c.want)
		}
	}
	if Decimals("JPY") != 0 || Decimals("gbp") != 2 || Decimals("kwd") != 2 {
		t.Error("an amount is said with the places its currency has")
	}
}
