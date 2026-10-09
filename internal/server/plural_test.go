package server

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// TestPlural pins down English pluralisation used in navigation labels,
// empty-state text and listing page headings. The old implementation simply
// appended "s" to every word that didn't already end in "s", producing output
// like "activitys".  This test ensures the consonant+y → ies rule is applied
// while preserving words ending in "s" and vowel+y forms.
func TestPlural(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"activity", "activities"},
		{"category", "categories"},
		{"party", "parties"},
		{"note", "notes"},
		{"person", "people"},
		{"message", "messages"},
		{"bus", "bus"},
		{"class", "class"},
		{"boy", "boys"},
		{"day", "days"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := schema.Plural(c.in); got != c.want {
				t.Errorf("schema.Plural(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	// Underscored names (used internally for type keys) get spaces too.
	t.Run("underscores", func(t *testing.T) {
		if got := schema.Plural("user_profile"); got != "user profiles" {
			t.Errorf("schema.Plural(%q) = %q, want %q", "user_profile", got, "user profiles")
		}
	})

	// A single-character word ending in y still gets +s (edge case).
	t.Run("single letter y", func(t *testing.T) {
		if got := schema.Plural("y"); got != "ys" {
			t.Errorf(`schema.Plural("y") = %q, want "ys"`, got)
		}
	})

	// A two-letter word consonant+y should become …ies.
	t.Run("two letter consonant y", func(t *testing.T) {
		if got := schema.Plural("by"); got != "bies" {
			t.Errorf(`schema.Plural("by") = %q, want "bies"`, got)
		}
	})

	// A two-letter word vowel+y should become +s.
	t.Run("two letter vowel y", func(t *testing.T) {
		if got := schema.Plural("ay"); got != "ays" {
			t.Errorf(`schema.Plural("ay") = %q, want "ays"`, got)
		}
	})
}
