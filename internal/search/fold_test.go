package search

import (
	"reflect"
	"testing"
)

// A search is read the way a person means it: case and accents do not
// matter, and a plural finds its one.
func TestWordsAreForgiving(t *testing.T) {
	if got := Words("Plumbers CAFÉ glass"); !reflect.DeepEqual(got, []string{"plumber", "cafe", "glass"}) {
		t.Errorf("Words = %v", got)
	}
	if fold("Crème brûlée") != "creme brulee" {
		t.Errorf("fold = %q", fold("Crème brûlée"))
	}
}

// Where words are found is a place in the text as it is, even where
// folding changed a letter's length, so marking them never cuts a letter.
func TestSpansAreInTheTextAsItIs(t *testing.T) {
	text := "İstanbul café, and the Café"
	spans := Spans(text, Words("cafe"))
	if len(spans) != 2 {
		t.Fatalf("spans = %v", spans)
	}
	for _, sp := range spans {
		if got := text[sp[0]:sp[1]]; got != "café" && got != "Café" {
			t.Errorf("span %v is %q", sp, got)
		}
	}
}

// A plural is looked for as its one, but marked as the word it is: the
// whole of groceries, not grocerie with its s left out (backlog 0547).
func TestSpansMarkWholeWords(t *testing.T) {
	marks := func(text, q string) []string {
		var out []string
		for _, sp := range Spans(text, Words(q)) {
			out = append(out, text[sp[0]:sp[1]])
		}
		return out
	}
	cases := []struct {
		text, q string
		want    []string
	}{
		{"Buy groceries — Note", "groceries", []string{"groceries"}},
		{"Groceries, then more groceries.", "groceries", []string{"Groceries", "groceries"}},
		{"Call the plumber and both plumbers", "plumbers", []string{"plumber", "plumbers"}},
		{"Two Cafés and a café", "cafes", []string{"Cafés", "café"}},
		{"Crème brûlées for the brûlée fan", "brulees creme", []string{"Crème", "brûlées", "brûlée"}},
		{"cafe\u0301s decomposed", "cafe", []string{"cafe\u0301s"}},
		{"the rosebuds bloom", "bud", []string{"rosebuds"}},
		{"日本語の文", "本", []string{"本"}},
	}
	for _, c := range cases {
		if got := marks(c.text, c.q); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Spans(%q, %q) marks %q, want %q", c.text, c.q, got, c.want)
		}
	}
}
