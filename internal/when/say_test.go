package when

import (
	"testing"
	"time"
)

// A time of day is said the GOV.UK way on the 12-hour clock: 2pm, not
// 2:00pm or 14:00hrs; midday and midnight, never 12pm.
func TestClockSaysATimeAsPeopleDo(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 9, 17, h, m, 0, 0, time.UTC) }
	for _, c := range []struct {
		t        time.Time
		twelve   string
		twenty4  string
		faceTwel string
	}{
		{at(14, 0), "2pm", "14:00", "2:00pm"},
		{at(17, 30), "5:30pm", "17:30", "5:30pm"},
		{at(9, 5), "9:05am", "09:05", "9:05am"},
		{at(12, 0), "midday", "12:00", "12:00pm"},
		{at(0, 0), "midnight", "00:00", "12:00am"},
		{at(0, 30), "12:30am", "00:30", "12:30am"},
		{at(12, 15), "12:15pm", "12:15", "12:15pm"},
	} {
		Hours24 = nil
		if got := Clock(c.t); got != c.twelve {
			t.Errorf("Clock(%s) = %q, want %q", c.t.Format("15:04"), got, c.twelve)
		}
		if got := Face(c.t); got != c.faceTwel {
			t.Errorf("Face(%s) = %q, want %q", c.t.Format("15:04"), got, c.faceTwel)
		}
		Hours24 = func() bool { return true }
		if got := Clock(c.t); got != c.twenty4 {
			t.Errorf("on the 24-hour clock, Clock(%s) = %q, want %q", c.t.Format("15:04"), got, c.twenty4)
		}
	}
	Hours24 = nil
}

// The clock is the person's choice, else their language's: English the
// 12-hour way, German and French the 24-hour way.
func TestTwentyFourFollowsTheChoiceThenTheLanguage(t *testing.T) {
	for _, c := range []struct {
		clock, lang string
		want        bool
	}{
		{"", "", false}, {"", "en", false}, {"", "en-GB", false}, {"", "de", true},
		{"", "fr-CA", true}, {"", "ko", false}, {"24", "en", true}, {"12", "de", false},
	} {
		if got := TwentyFour(c.clock, c.lang); got != c.want {
			t.Errorf("TwentyFour(%q, %q) = %v, want %v", c.clock, c.lang, got, c.want)
		}
	}
}

// A day planned for is said as a person plans by it: Today, Tomorrow,
// Yesterday, else its weekday and date, the year only when another.
// Nothing makes a person count days ("In 4 days").
func TestRelativeSaysADayToPlanBy(t *testing.T) {
	Hours24 = nil
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC) // a Thursday
	for _, c := range []struct{ v, want string }{
		{"2026-09-17T08:00:00Z", "Today at 8am"},
		{"2026-09-17T14:30:00Z", "Today at 2:30pm"},
		{"2026-09-16T23:59:00Z", "Yesterday at 11:59pm"},
		{"2026-09-18T12:00:00Z", "Tomorrow at midday"},
		{"2026-09-15T13:31:00Z", "Tue 15 Sep at 1:31pm"},
		{"2026-09-19T00:00:00Z", "Sat 19 Sep"},
		{"2026-09-23T06:00:00Z", "Wed 23 Sep at 6am"},
		{"2025-12-25T18:00:00Z", "Thu 25 Dec 2025 at 6pm"},
		{"2026-09-17T00:00:00Z", "Today"},
		{"2026-09-18T00:00:00Z", "Tomorrow"},
		{"2026-09-16T00:00:00Z", "Yesterday"},
		{"2025-03-15T00:00:00Z", "Sat 15 Mar 2025"},
		{"not a date", "not a date"},
		{"", ""},
	} {
		if got := Relative(c.v, now); got != c.want {
			t.Errorf("Relative(%q) = %q, want %q", c.v, got, c.want)
		}
	}
}

// A moment is said in the reader's zone: late on the 17th in UTC is
// already the 18th in Tokyo, and a day alone is the same day everywhere.
func TestRelativeIsSaidInTheReadersZone(t *testing.T) {
	Hours24 = nil
	tokyo := time.FixedZone("Tokyo", 9*3600)
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, tokyo)
	if got := Relative("2026-09-17T20:00:00Z", now); got != "Tomorrow at 5am" {
		t.Errorf("got %q, want Tomorrow at 5am", got)
	}
	if got := Relative("2026-09-18T00:00:00Z", now); got != "Tomorrow" {
		t.Errorf("a day alone is the day it is, got %q", got)
	}
}

// When something happened is said by how long ago while that is short,
// then by its date.
func TestAgoSaysWhenSomethingHappened(t *testing.T) {
	Hours24 = nil
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)
	for _, c := range []struct {
		t    time.Time
		want string
	}{
		{now.Add(-time.Hour), "Today at 9:30am"},
		{now.Add(time.Minute), "Today at 10:31am"},
		{time.Date(2026, 9, 16, 21, 0, 0, 0, time.UTC), "Yesterday at 9pm"},
		{time.Date(2026, 9, 14, 6, 0, 0, 0, time.UTC), "3 days ago"},
		{time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC), "1 Sep"},
		{time.Date(2025, 9, 1, 6, 0, 0, 0, time.UTC), "1 Sep 2025"},
	} {
		if got := Ago(c.t, now); got != c.want {
			t.Errorf("Ago(%s) = %q, want %q", c.t, got, c.want)
		}
	}
}

// The full date is there for words that leave it out, and a <time> holds
// the value a machine reads: the date alone for a day.
func TestFullAndMachine(t *testing.T) {
	Hours24 = nil
	if got := Full("2026-10-05T00:00:00Z"); got != "Monday 5 October 2026" {
		t.Errorf("Full of a day = %q", got)
	}
	if got := Machine("2026-10-05T00:00:00Z"); got != "2026-10-05" {
		t.Errorf("Machine of a day = %q", got)
	}
	if got := Machine("2026-10-05T14:00:00Z"); got != "2026-10-05T14:00:00Z" {
		t.Errorf("Machine of a moment = %q", got)
	}
	for words, out := range map[string]bool{"Today at 2pm": true, "3 days ago": true, "Fri 9 Oct": false, "Tomorrow": true} {
		if LeavesDateOut(words) != out {
			t.Errorf("LeavesDateOut(%q) should be %v", words, out)
		}
	}
}

// What Text says reads back as the same value on either clock.
func TestTextReadsBackOnEitherClock(t *testing.T) {
	defer func() { Hours24 = nil }()
	for _, h24 := range []bool{false, true} {
		Hours24 = func() bool { return h24 }
		for _, v := range []string{"2026-09-19T14:00:00Z", "2026-09-19T11:30:00Z", "2026-09-19T00:00:00Z"} {
			ts, day, ok := Parse(Text(v), time.Now())
			if !ok || Store(ts, day) != v {
				t.Errorf("24-hour %v: Text(%s) = %q reads back as %s", h24, v, Text(v), Store(ts, day))
			}
		}
	}
}
