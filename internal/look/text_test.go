package look

import (
	"strings"
	"testing"
)

// What a page says is read passage by passage under its heading, its
// words spaced as a reader hears them, what is hidden from everyone left
// out and what is hidden only from the eye read.
func TestAPageSaysWhatItSays(t *testing.T) {
	o, err := Page(`<!doctype html><html lang="en"><head><title>Buy paint</title></head><body><main>
<h1>Buy paint</h1>
<p class="lede"><span class="sw-badge">Done</span><time>Fri 9 Oct</time><span class="sw-visually-hidden">, for Ana</span><span aria-hidden="true">✓</span></p>
<ul><li><p>Sage for the hall.</p></li><li>Second</li></ul>
<h2>Activity</h2><p hidden>Not shown</p><dl><dt>Added</dt><dd>Today at 2pm</dd></dl>
</main></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range o.Text {
		got = append(got, p.Under+": "+p.Text)
	}
	want := "Buy paint: Done Fri 9 Oct, for Ana|Buy paint: Sage for the hall.|Buy paint: Second|Activity: Added|Activity: Today at 2pm"
	if strings.Join(got, "|") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, "|"), want)
	}
}
