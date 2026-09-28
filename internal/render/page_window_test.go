package render

import (
	"fmt"
	"testing"
)

// The pages shown are the first, the last and one either side of this one,
// never more than seven with gaps, and a gap always stands for two or more.
func TestThePageWindowIsShortAndHidesNoSinglePage(t *testing.T) {
	for _, c := range []struct {
		p, n int
		want string
	}{
		{1, 1, "[1]"},
		{5, 12, "[1 0 4 5 6 0 12]"},
		{4, 12, "[1 2 3 4 5 0 12]"},
		{1, 12, "[1 2 0 12]"},
		{12, 12, "[1 0 11 12]"},
		{3, 5, "[1 2 3 4 5]"},
	} {
		if got := fmt.Sprint(pageWindow(c.p, c.n)); got != c.want {
			t.Errorf("page %d of %d shows %s, want %s", c.p, c.n, got, c.want)
		}
	}
}
