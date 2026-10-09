package when

import (
	"strings"
	"testing"
)

// A month shows each later time a repeat falls in it, and no more: not
// the time it is due, not a day outside the month, not past its end, and
// never more than asked, however long it goes on.
func TestOccurrencesAreTheMonthsAndNoMore(t *testing.T) {
	t.Parallel()
	cases := []struct {
		repeat, from, first, last string
		most                      int
		want                      string
	}{
		{"FREQ=WEEKLY;BYDAY=TU", "2026-09-01T00:00:00Z", "2026-09-01", "2026-09-30", 31, "2026-09-08 2026-09-15 2026-09-22 2026-09-29"},
		{"FREQ=WEEKLY;BYDAY=TU", "2026-08-04T00:00:00Z", "2026-09-01", "2026-09-30", 31, "2026-09-01 2026-09-08 2026-09-15 2026-09-22 2026-09-29"},
		{"FREQ=MONTHLY;BYMONTHDAY=31", "2026-01-31T00:00:00Z", "2026-02-01", "2026-02-28", 31, "2026-02-28"},
		{"FREQ=DAILY;UNTIL=20260903", "2026-09-01T00:00:00Z", "2026-09-01", "2026-09-30", 31, "2026-09-02 2026-09-03"},
		{"FREQ=DAILY", "2026-09-01T00:00:00Z", "2026-09-01", "2026-09-30", 5, "2026-09-02 2026-09-03 2026-09-04 2026-09-05 2026-09-06"},
		{"FREQ=WEEKLY;BYDAY=TU", "2026-10-06T00:00:00Z", "2026-09-01", "2026-09-30", 31, ""},
		{"FREQ=WEEKLY;BYDAY=TU", "", "2026-09-01", "2026-09-30", 31, ""},
	}
	for _, c := range cases {
		var days []string
		for _, v := range Occurrences(c.repeat, c.from, c.first, c.last, c.most) {
			days = append(days, v[:10])
		}
		if got := strings.Join(days, " "); got != c.want {
			t.Errorf("%s from %s in %s..%s: %q, want %q", c.repeat, c.from, c.first, c.last, got, c.want)
		}
	}
	// Every day since 2020 is every day of September, found from 2020 on.
	if n := len(Occurrences("FREQ=DAILY", "2020-01-01T00:00:00Z", "2026-09-01", "2026-09-30", 31)); n != 30 {
		t.Errorf("every day since 2020 is every day of the month, got %d", n)
	}
}
