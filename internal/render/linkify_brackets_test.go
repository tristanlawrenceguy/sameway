package render

import "testing"

// A Markdown link whose words hold a pair of brackets, a record called
// "Seeds [draft]", is a link by those words, not its syntax shown.
func TestALinkByATitleWithBracketsIsALink(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"See [Seeds [draft]](/t/note/seeds1).":   "See Seeds [draft].",
		"Filed [[Work] plan](/t/note/seeds1)":    "Filed [Work] plan",
		"Filed [Seeds](/t/note/seeds1) as asked": "Filed Seeds as asked",
	} {
		if got := LinkWords(in, func(string) string { return "" }); got != want {
			t.Errorf("LinkWords(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}
