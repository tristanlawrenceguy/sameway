package when

import (
	"testing"
	"time"
)

// The ways people write a day or a moment are all read, relative to a
// Thursday in September; what is not understood is refused, not guessed.
func TestParseReadsWhatPeopleWrite(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC) // a Thursday
	cases := []struct {
		in   string
		want string // as stored
		ok   bool
	}{
		{"today", "2026-09-17T00:00:00Z", true},
		{"tomorrow", "2026-09-18T00:00:00Z", true},
		{"yesterday", "2026-09-16T00:00:00Z", true},
		{"friday", "2026-09-18T00:00:00Z", true},
		{"Thursday", "2026-09-17T00:00:00Z", true},
		{"next thursday", "2026-09-24T00:00:00Z", true},
		{"next Friday", "2026-09-18T00:00:00Z", true},
		{"last monday", "2026-09-14T00:00:00Z", true},
		{"next week", "2026-09-24T00:00:00Z", true},
		{"next month", "2026-10-17T00:00:00Z", true},
		{"19 sep", "2026-09-19T00:00:00Z", true},
		{"19 Sep 2026", "2026-09-19T00:00:00Z", true},
		{"Sep 19", "2026-09-19T00:00:00Z", true},
		{"September 19th, 2026", "2026-09-19T00:00:00Z", true},
		{"19 september", "2026-09-19T00:00:00Z", true},
		{"the 3rd", "2026-10-03T00:00:00Z", true},
		{"on the 20th", "2026-09-20T00:00:00Z", true},
		{"19/9", "2026-09-19T00:00:00Z", true},
		{"19/9/2026", "2026-09-19T00:00:00Z", true},
		{"9/19/26", "2026-09-19T00:00:00Z", true},
		{"2026-09-19", "2026-09-19T00:00:00Z", true},
		{"2026-09-19T00:00:00Z", "2026-09-19T00:00:00Z", true},
		{"2026-09-19T14:30:00Z", "2026-09-19T14:30:00Z", true},
		{"2026-09-19 14:30", "2026-09-19T14:30:00Z", true},
		{"tomorrow 2pm", "2026-09-18T14:00:00Z", true},
		{"tomorrow at 2 pm", "2026-09-18T14:00:00Z", true},
		{"tomorrow at 9", "2026-09-18T09:00:00Z", true},
		{"fri 14:00", "2026-09-18T14:00:00Z", true},
		{"2:30pm", "2026-09-17T14:30:00Z", true},
		{"12am", "2026-09-17T00:00:00Z", true},
		{"noon", "2026-09-17T12:00:00Z", true},
		{"19 sep midnight", "2026-09-19T00:00:00Z", true},
		{"in 3 days", "2026-09-20T00:00:00Z", true},
		{"2 weeks", "2026-10-01T00:00:00Z", true},
		{"3 days ago", "2026-09-14T00:00:00Z", true},
		{"in 3 hours", "2026-09-17T13:30:00Z", true},
		{"now", "2026-09-17T10:30:00Z", true},
		{"+7d", "2026-09-24T10:30:00Z", true},
		{"-1w", "2026-09-10T10:30:00Z", true},
		{"Sat 19 Sep 2026, 14:00", "2026-09-19T14:00:00Z", true},
		{"Sat 19 Sep 2026", "2026-09-19T00:00:00Z", true},
		{"sep", "2026-09-01T00:00:00Z", true},
		{"", "", false},
		{"sometime", "", false},
		{"19 sep or so", "", false},
		{"next", "", false},
	}
	for _, c := range cases {
		got, day, ok := Parse(c.in, now)
		if ok != c.ok {
			t.Errorf("Parse(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if s := Store(got, day); s != c.want {
			t.Errorf("Parse(%q) = %s (day %v), want %s", c.in, s, day, c.want)
		}
	}
}

// A stored value reads as a person would say it, and reads back.
func TestTextRoundTrips(t *testing.T) {
	if got := Text("2026-09-19T00:00:00Z"); got != "Sat 19 Sep 2026" {
		t.Errorf("a day reads as its date, got %q", got)
	}
	moment := time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)
	if got, want := Text("2026-09-19T14:00:00Z"), moment.Local().Format("Mon 2 Jan 2006, 15:04"); got != want {
		t.Errorf("a moment reads in local time, got %q want %q", got, want)
	}
	if got := Text("whatever was typed"); got != "whatever was typed" {
		t.Errorf("words that are not a value come back as they are, got %q", got)
	}
	for _, v := range []string{"2026-09-19T00:00:00Z", "2026-09-19T14:00:00Z"} {
		ts, day, ok := Parse(Text(v), time.Now())
		if !ok || Store(ts, day) != v {
			t.Errorf("Text(%s) = %q should read back as itself, got %s", v, Text(v), Store(ts, day))
		}
	}
}
