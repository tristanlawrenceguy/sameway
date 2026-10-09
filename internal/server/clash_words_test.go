package server

import (
	"strings"
	"testing"
)

// What differs is found word by word, a change is one run, and reading
// the parts one way gives each version back whole.
func TestClashPartsReadBackAsEachVersion(t *testing.T) {
	t.Parallel()
	for _, c := range [][2]string{
		{"Tea and toast at nine.", "Tea and cake at nine."},
		{"Flour, water, salt.\nBake at once.", "Flour, water, salt.\nLeave overnight."},
		{"one", "one two three"},
		{"", "all new"},
		{"a b c d", "x y"},
	} {
		parts := clashParts(c[0], c[1])
		var page, this strings.Builder
		for _, p := range parts {
			m := p.(map[string]any)
			text := m["text"].(string)
			switch m["in"] {
			case "both":
				page.WriteString(text)
				this.WriteString(text)
			case "page":
				page.WriteString(text)
			case "this":
				this.WriteString(text)
			}
		}
		if strings.Join(strings.Fields(page.String()), " ") != strings.Join(strings.Fields(c[0]), " ") || this.String() != c[1] {
			t.Errorf("%q / %q read back as %q / %q: %v", c[0], c[1], page.String(), this.String(), parts)
		}
	}
	parts := clashParts("Tea and toast at nine.", "Tea and cake at nine.")
	if len(parts) != 4 || parts[1].(map[string]any)["text"] != "toast" || parts[2].(map[string]any)["text"] != "cake" {
		t.Errorf("one word changed is one change: %v", parts)
	}
	if clashParts("same", "same") != nil {
		t.Error("the same text has nothing to show")
	}
}
