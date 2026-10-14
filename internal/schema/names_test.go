package schema

import "testing"

// Names with commas of their own are set apart by semicolons, to the
// last; names with none keep commas.
func TestNamesWithCommasAreSetApart(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		names     []string
		row, said string
	}{
		{nil, "", ""},
		{[]string{"Email Sarah"}, "Email Sarah", "Email Sarah"},
		{[]string{"Email Sarah", "Call mum", "Pay rent"}, "Email Sarah, Call mum, Pay rent", "Email Sarah, Call mum and Pay rent"},
		{[]string{"Running, 5 miles", "Running, 3 miles"}, "Running, 5 miles; Running, 3 miles", "Running, 5 miles; and Running, 3 miles"},
		{[]string{"Bob", "Silva, Ana", "Kim"}, "Bob; Silva, Ana; Kim", "Bob; Silva, Ana; and Kim"},
	} {
		if got := Names(c.names); got != c.row {
			t.Errorf("Names(%q) = %q, want %q", c.names, got, c.row)
		}
		if got := AndNames(c.names); got != c.said {
			t.Errorf("AndNames(%q) = %q, want %q", c.names, got, c.said)
		}
	}
}
