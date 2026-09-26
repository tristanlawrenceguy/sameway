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
