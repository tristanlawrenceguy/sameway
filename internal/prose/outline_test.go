package prose

import (
	"strings"
	"testing"
)

// Headings take their place in the page's outline: the shallowest written
// is at the base, whatever it was written as, and none skips a level.
func TestHeadingsNeverSkipALevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"## Plan\n\n### Step", []string{"<h2>Plan</h2>", "<h3>Step</h3>"}},
		{"# Plan\n\n### Step", []string{"<h2>Plan</h2>", "<h3>Step</h3>"}},
		{"# A\n\n## B\n\n# C", []string{"<h2>A</h2>", "<h3>B</h3>", "<h2>C</h2>"}},
	} {
		got := string(Render(c.in, 2))
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%q at base 2 should give %s, got %s", c.in, w, got)
			}
		}
	}
}
