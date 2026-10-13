package schema

import "testing"

// TestUnchosen: a default, or no on a yes-or-no, was nobody's choice;
// anything else was.
func TestUnchosen(t *testing.T) {
	t.Parallel()
	method := Field{Type: "enum", Values: []string{"GET", "POST"}, Default: "POST"}
	archived := Field{Type: "bool"}
	many := Field{Type: "int", Default: 10}
	cases := []struct {
		f    Field
		v    any
		want bool
	}{
		{method, "POST", true}, {method, "GET", false},
		{archived, false, true}, {archived, true, false},
		{many, 10, true}, {many, float64(10), true}, {many, 3, false},
		{Field{Type: "string"}, "words", false}, {Field{Type: "string"}, nil, true},
	}
	for _, c := range cases {
		if got := c.f.Unchosen(c.v); got != c.want {
			t.Errorf("%+v Unchosen(%v) = %v, want %v", c.f, c.v, got, c.want)
		}
	}
}
