package fold_test

import (
	"slices"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
)

// Finding reads letters as a person without them on their keyboard types
// them: accents, case and width off, and the letters Unicode does not
// count as accented (ß, ł, ø, dotless ı) spelled plainly.
func TestStringFindsWhatAPersonTypes(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"Straße":       "strasse",
		"STRASSE":      "strasse",
		"Łódź":         "lodz",
		"København":    "kobenhavn",
		"Diyarbakır":   "diyarbakir",
		"İstanbul":     "istanbul",
		"ＣＡＦＥ":         "cafe",
		"oﬃce":         "office",
		"Ægir":         "aegir",
		"Mum’s – trip": "mum's - trip",
		"Café":        "cafe", // é stored as e and an accent
		"東京":           "東京",
		"שלום":         "שלום",
	} {
		if got := fold.String(in); got != want {
			t.Errorf("String(%q) = %q, want %q", in, got, want)
		}
	}
}

// One name is one name whatever form its letters are stored in and in any
// case; accents still make two names, and a curly apostrophe is heard as a
// straight one.
func TestKeyIsOneNameInAnyFormOrCase(t *testing.T) {
	t.Parallel()
	same := [][2]string{
		{"Café", "Café"}, // é as one letter, and as e and an accent
		{"STRASSE", "straße"},
		{"Buy  groceries ", "buy groceries"},
		{"Mum’s birthday", "Mum's birthday"},
		{"ΣΟΦΟΣ", "σοφος"},
	}
	for _, p := range same {
		if !fold.Same(p[0], p[1]) {
			t.Errorf("%q and %q should be one name: %q, %q", p[0], p[1], fold.Key(p[0]), fold.Key(p[1]))
		}
	}
	for _, p := range [][2]string{{"Cafe", "Café"}, {"Item 2", "Item 10"}, {"ＣＡＦＥ", "CAFE"}} {
		if fold.Same(p[0], p[1]) {
			t.Errorf("%q and %q are two names", p[0], p[1])
		}
	}
}

// A list in order is as a person reads it: Émile with the E's, case only
// breaking a tie, and numbers by their value.
func TestCompareSortsAsAPersonReads(t *testing.T) {
	t.Parallel()
	got := []string{"Zoë", "item 10", "Émile", "apple", "Item 2", "Banana", "emma"}
	slices.SortStableFunc(got, fold.Compare)
	want := []string{"apple", "Banana", "Émile", "emma", "Item 2", "item 10", "Zoë"}
	if !slices.Equal(got, want) {
		t.Errorf("sorted %q, want %q", got, want)
	}
}
