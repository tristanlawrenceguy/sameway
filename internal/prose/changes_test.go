package prose

import (
	"strings"
	"testing"
)

// A formatting change is said in words, and different words are not one.
func TestAFormattingChangeIsSaidInWords(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ from, to, want string }{
		{"the garden opens", "the **garden** opens", "Make it bold"},
		{"Plans", "## Plans", "Make it a heading"},
		{"milk\neggs", "- milk\n- eggs", "Make it a list"},
		{"the report", "[the report](https://example.com)", "Add a link"},
		{"*quiet*", "quiet", "Take the italics off"},
	} {
		got, only := FormatChange(c.from, c.to)
		if !only || got != c.want {
			t.Errorf("%q to %q: got %q %v, want %q", c.from, c.to, got, only, c.want)
		}
	}
	if _, only := FormatChange("the cat", "the dog"); only {
		t.Error("different words are not a formatting change")
	}
}

// The passage is highlighted in the writing as it reads, with no Markdown
// showing; where it cannot be one stretch, the writing is shown unmarked.
func TestThePassageIsMarkedInTheWritingAsItReads(t *testing.T) {
	t.Parallel()
	got := string(Marked("We **all** agreed that the garden opens.", "agreed that", "Change starts", "change ends", 3))
	if !strings.Contains(got, `<span class="sw-prose__changed"><span class="sw-visually-hidden">Change starts: </span>agreed that<span class="sw-visually-hidden">, change ends,</span></span> the garden`) || strings.Contains(got, "**") || strings.Contains(got, "<mark") {
		t.Errorf("marked as it reads: %s", got)
	}
	if got := string(Marked("## Plans\n\nSome text", "## Plans", "", "", 3)); !strings.Contains(got, "<h3") || !strings.Contains(got, "sw-prose__changed") {
		t.Errorf("a heading stays a heading, marked: %s", got)
	}
	if got := string(Marked("One.\n\nTwo.", "One.\n\nTwo", "", "", 3)); strings.Contains(got, "sw-prose__changed") || strings.Contains(got, "\uE000") {
		t.Errorf("across paragraphs, shown unmarked: %s", got)
	}
}
