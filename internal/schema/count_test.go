package schema

import "testing"

// A count says what it counts, one or many.
func TestCount(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n          int
		name, want string
	}{{1, "task", "1 task"}, {3, "task", "3 tasks"}, {0, "task", "0 tasks"}, {2, "person", "2 people"}, {2, "call_log", "2 call logs"}, {4, "entry", "4 entries"}} {
		if got := Count(c.n, c.name); got != c.want {
			t.Errorf("Count(%d, %s) = %q, want %q", c.n, c.name, got, c.want)
		}
	}
}
