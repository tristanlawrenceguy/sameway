package when

import (
	"strings"
	"testing"
	"time"
)

// The ways people say how often are read, kept as a small RRULE, and said
// back in words that read as the same repeat; what is not understood is
// refused with what it must be.
func TestParseRepeatReadsWhatPeopleSay(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC) // a Thursday
	cases := []struct{ in, stored, said string }{
		{"every day", "FREQ=DAILY", "every day"},
		{"Daily", "FREQ=DAILY", "every day"},
		{"every 3 days", "FREQ=DAILY;INTERVAL=3", "every 3 days"},
		{"every other day", "FREQ=DAILY;INTERVAL=2", "every 2 days"},
		{"every weekday", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", "every weekday"},
		{"every weekend", "FREQ=WEEKLY;BYDAY=SA,SU", "every Saturday and Sunday"},
		{"every Tuesday", "FREQ=WEEKLY;BYDAY=TU", "every Tuesday"},
		{"Repeats every tues.", "FREQ=WEEKLY;BYDAY=TU", "every Tuesday"},
		{"every friday and monday", "FREQ=WEEKLY;BYDAY=MO,FR", "every Monday and Friday"},
		{"every mon, wed and fri", "FREQ=WEEKLY;BYDAY=MO,WE,FR", "every Monday, Wednesday and Friday"},
		{"every week", "FREQ=WEEKLY", "every week"},
		{"weekly", "FREQ=WEEKLY", "every week"},
		{"every 2 weeks", "FREQ=WEEKLY;INTERVAL=2", "every 2 weeks"},
		{"fortnightly", "FREQ=WEEKLY;INTERVAL=2", "every 2 weeks"},
		{"every 2 weeks on Tuesday", "FREQ=WEEKLY;INTERVAL=2;BYDAY=TU", "every 2 weeks on Tuesday"},
		{"every month", "FREQ=MONTHLY", "every month"},
		{"every month on the 1st", "FREQ=MONTHLY;BYMONTHDAY=1", "every month on the 1st"},
		{"monthly on the 31st", "FREQ=MONTHLY;BYMONTHDAY=31", "every month on the 31st"},
		{"every month on the last day", "FREQ=MONTHLY;BYMONTHDAY=-1", "every month on the last day"},
		{"every 1st of the month", "FREQ=MONTHLY;BYMONTHDAY=1", "every month on the 1st"},
		{"every 15th", "FREQ=MONTHLY;BYMONTHDAY=15", "every month on the 15th"},
		{"every last day of the month", "FREQ=MONTHLY;BYMONTHDAY=-1", "every month on the last day"},
		{"every 3 months on the 22nd", "FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=22", "every 3 months on the 22nd"},
		{"every year", "FREQ=YEARLY", "every year"},
		{"annually", "FREQ=YEARLY", "every year"},
		{"every year on 29 Feb", "FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=29", "every year on 29 Feb"},
		{"every year on March 3rd", "FREQ=YEARLY;BYMONTH=3;BYMONTHDAY=3", "every year on 3 Mar"},
		{"every day until 1 Mar", "FREQ=DAILY;UNTIL=20270301", "every day until Mon 1 Mar 2027"},
		{"every day until 30 Sep", "FREQ=DAILY;UNTIL=20260930", "every day until Wed 30 Sep 2026"},
		{"every Tuesday until 1 Mar 2027", "FREQ=WEEKLY;BYDAY=TU;UNTIL=20270301", "every Tuesday until Mon 1 Mar 2027"},
		{"FREQ=WEEKLY;BYDAY=TU", "FREQ=WEEKLY;BYDAY=TU", "every Tuesday"},
		{"never", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		got, ok := ParseRepeat(c.in, now)
		if !ok || got != c.stored {
			t.Errorf("%q: stored %q (ok %v), want %q; %s", c.in, got, ok, c.stored, RepeatWhy(c.in, now))
			continue
		}
		if c.stored == "" {
			continue
		}
		if said := RepeatText(got); said != c.said {
			t.Errorf("%q: said %q, want %q", c.in, said, c.said)
		}
		// What is said back reads as the same repeat.
		if again, _ := ParseRepeat(RepeatText(got), now); again != got {
			t.Errorf("%q: said back as %q, which reads as %q", c.in, RepeatText(got), again)
		}
	}
}

func TestParseRepeatRefusesWithWhatItMustBe(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)
	cases := map[string]string{
		"sometimes":                    "must say how often",
		"every blue moon":              "must say how often",
		"every month on the 32nd":      "from the 1st to the 31st",
		"every year on 30 Feb":         "must be on a real day: 30 feb is not one",
		"every 2 weeks on mon and thu": "one day of the week",
		"every day until someday":      "must end on a real day",
		"every weekday on monday":      "must say how often",
		"FREQ=MONTHLY;BYSETPOS=-1":     "must say how often",
	}
	for in, want := range cases {
		if _, ok := ParseRepeat(in, now); ok {
			t.Errorf("%q was read; it should be refused", in)
		}
		if why := RepeatWhy(in, now); !strings.Contains(why, want) {
			t.Errorf("%q: why %q, want it to say %q", in, why, want)
		}
	}
	if RepeatWhy("every day", now) != "" {
		t.Error("words that are read have no why")
	}
}

// The next time falls on the schedule counted from the day it was due:
// the 31st on the last day of shorter months, 29 Feb on 28 Feb in other
// years, weekdays over the weekend, and not after it ends.
func TestNextKeepsTheSchedule(t *testing.T) {
	now := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		repeat, from, want string
		ok                 bool
	}{
		{"FREQ=DAILY", "2026-01-10T00:00:00Z", "2026-01-11T00:00:00Z", true},
		{"FREQ=DAILY;INTERVAL=3", "2026-01-12T00:00:00Z", "2026-01-15T00:00:00Z", true},
		// Overdue: the next one still to come on its schedule, not today.
		{"FREQ=DAILY", "2026-01-05T00:00:00Z", "2026-01-11T00:00:00Z", true},
		{"FREQ=WEEKLY;INTERVAL=2", "2026-01-01T00:00:00Z", "2026-01-15T00:00:00Z", true},
		{"FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", "2026-01-16T00:00:00Z", "2026-01-19T00:00:00Z", true}, // Friday to Monday
		{"FREQ=WEEKLY;BYDAY=TU", "2026-01-13T00:00:00Z", "2026-01-20T00:00:00Z", true},
		{"FREQ=WEEKLY;INTERVAL=2;BYDAY=TU", "2026-01-13T00:00:00Z", "2026-01-27T00:00:00Z", true},
		{"FREQ=MONTHLY;BYMONTHDAY=31", "2026-01-31T00:00:00Z", "2026-02-28T00:00:00Z", true},
		{"FREQ=MONTHLY;BYMONTHDAY=31", "2026-02-28T00:00:00Z", "2026-03-31T00:00:00Z", true},
		{"FREQ=MONTHLY;BYMONTHDAY=31", "2026-03-31T00:00:00Z", "2026-04-30T00:00:00Z", true},
		{"FREQ=MONTHLY;BYMONTHDAY=-1", "2026-01-31T00:00:00Z", "2026-02-28T00:00:00Z", true},
		{"FREQ=MONTHLY;BYMONTHDAY=1", "2026-01-15T00:00:00Z", "2026-02-01T00:00:00Z", true},
		{"FREQ=MONTHLY;BYMONTHDAY=20", "2026-01-15T00:00:00Z", "2026-01-20T00:00:00Z", true},
		{"FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=29", "2028-02-29T00:00:00Z", "2029-02-28T00:00:00Z", true},
		{"FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=29", "2031-02-28T00:00:00Z", "2032-02-29T00:00:00Z", true},
		{"FREQ=DAILY;UNTIL=20260112", "2026-01-11T00:00:00Z", "2026-01-12T00:00:00Z", true},
		{"FREQ=DAILY;UNTIL=20260112", "2026-01-12T00:00:00Z", "", false},
		{"FREQ=DAILY", "2026-01-12T09:30:00Z", "2026-01-13T09:30:00Z", true},
		// Nothing in the field: from today.
		{"FREQ=DAILY", "", "2026-01-11T00:00:00Z", true},
		{"not a rule", "2026-01-10T00:00:00Z", "", false},
	}
	for _, c := range cases {
		got, ok := Next(c.repeat, c.from, now.In(time.UTC))
		if got != c.want || ok != c.ok {
			t.Errorf("%s from %s: %q %v, want %q %v", c.repeat, c.from, got, ok, c.want, c.ok)
		}
	}
}

// A time of day stays the time on the wall across a change of clocks.
func TestNextKeepsTheTimeAcrossSummerTime(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("no time zone data here")
	}
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, london)
	from := time.Date(2026, 3, 28, 9, 0, 0, 0, london) // the day before the clocks go forward
	got, _ := Next("FREQ=DAILY", Store(from, false), now)
	ts, _ := time.Parse(time.RFC3339, got)
	if l := ts.In(london); l.Hour() != 9 || l.Day() != 29 {
		t.Errorf("every day at 09:00 is 09:00 after the clocks change, got %v", l)
	}
}

// A monthly or yearly repeat without its day takes the day it was due, so
// the 31st does not become the 28th for good after February.
func TestPinKeepsTheDay(t *testing.T) {
	if got := Pin("FREQ=MONTHLY", "2026-01-31T00:00:00Z"); got != "FREQ=MONTHLY;BYMONTHDAY=31" {
		t.Errorf("monthly from the 31st: %q", got)
	}
	if got := Pin("FREQ=YEARLY", "2028-02-29T00:00:00Z"); got != "FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=29" {
		t.Errorf("yearly from 29 Feb: %q", got)
	}
	if got := Pin("FREQ=WEEKLY;BYDAY=TU", "2026-01-31T00:00:00Z"); got != "FREQ=WEEKLY;BYDAY=TU" {
		t.Errorf("a weekly repeat is kept as it is: %q", got)
	}
}

// Put off by five minutes, a repeat keeps to its own time of day, and
// lets it go once the next time is set.
func TestKeepTimeHoldsTheTimeWhilePutOff(t *testing.T) {
	now := time.Date(2026, 1, 10, 7, 1, 0, 0, time.Local)
	held := KeepTime("FREQ=DAILY", Store(time.Date(2026, 1, 10, 7, 0, 0, 0, time.Local), false))
	if RepeatText(held) != "every day" {
		t.Errorf("a held time is not said, got %q", RepeatText(held))
	}
	got, _ := Next(held, Store(time.Date(2026, 1, 10, 7, 5, 0, 0, time.Local), false), now)
	if want := Store(time.Date(2026, 1, 11, 7, 0, 0, 0, time.Local), false); got != want {
		t.Errorf("five more minutes today, 07:00 tomorrow: got %s, want %s", got, want)
	}
	if DropTime(held) != "FREQ=DAILY" {
		t.Errorf("let go, got %q", DropTime(held))
	}
}
