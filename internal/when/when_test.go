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
		{"2026-09-19 2pm", "2026-09-19T14:00:00Z", true},
		{"2026-09-19 at 9:15am", "2026-09-19T09:15:00Z", true},
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
		// Dots, as many write a date or a time.
		{"19.9", "2026-09-19T00:00:00Z", true},
		{"19.9.2026", "2026-09-19T00:00:00Z", true},
		{"19.09.26", "2026-09-19T00:00:00Z", true},
		{"2026/09/19", "2026-09-19T00:00:00Z", true},
		{"tomorrow 14.30", "2026-09-18T14:30:00Z", true},
		{"9.30", "2026-09-17T09:30:00Z", true},
		{"2.30pm", "2026-09-17T14:30:00Z", true},
		{"Sat. 19 Sep.", "2026-09-19T00:00:00Z", true},
		{"the 31st", "2026-10-31T00:00:00Z", true},
		// A date or time that does not exist is asked again, not rolled on.
		{"31/2", "", false},
		{"30 feb", "", false},
		{"32 sep", "", false},
		{"sep 45", "", false},
		{"19/13", "", false},
		{"25:00", "", false},
		{"14:75", "", false},
		{"13pm", "", false},
		// A weekday that is not the date's.
		{"Fri 19 Sep", "", false},
		{"19 Sep, Friday", "", false},
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

// Words not read say why, as what they must be.
func TestWhySaysWhatItMustBe(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)
	for in, want := range map[string]string{
		"sometime":   "must be a day, like 19 Sep, next Friday or tomorrow 2pm",
		"31 feb":     "must be a real day: 31 feb is not one",
		"25:00":      "must be a real time: 25:00 is not one",
		"Fri 19 Sep": "must be one day: 19 Sep 2026 is a Saturday, not a Friday",
		"19 Sep":     "",
	} {
		if got := Why(in, now); got != want {
			t.Errorf("Why(%q) = %q, want %q", in, got, want)
		}
	}
	if _, _, ok := Parse("midday", now); !ok {
		t.Error("midday is read, as noon is")
	}
}

// Relative formats stored moments and days as natural-language phrasing.
func TestRelativeFormats(t *testing.T) {
	// A Thursday at mid-morning — all comparisons relative to this.
	now := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		v    string
		want string
	}{
		// Today (diff == 0).
		{"2026-09-17T08:00:00Z", "Today at 8:00am"},
		{"2026-09-17T14:30:00Z", "Today at 2:30pm"},
		// Yesterday (diff == -1).
		{"2026-09-16T09:00:00Z", "Yesterday at 9:00am"},
		{"2026-09-16T23:59:00Z", "Yesterday at 11:59pm"},
		// Tomorrow (diff == 1).
		{"2026-09-18T07:15:00Z", "Tomorrow at 7:15am"},
		// Within past week (diff -2..-6).
		{"2026-09-15T13:31:00Z", "2 days ago at 1:31pm"},
		{"2026-09-14T06:00:00Z", "3 days ago at 6:00am"},
		{"2026-09-11T22:00:00Z", "6 days ago at 10:00pm"},
		// Within future week (diff 2..6).
		{"2026-09-19T13:31:00Z", "In 2 days at 1:31pm"},
		{"2026-09-23T06:00:00Z", "In 6 days at 6:00am"},
		// Older same-year (diff >= 7 or <= -7).
		{"2026-08-01T15:45:00Z", "Sat 1 Aug at 3:45pm"},
		{"2026-01-01T00:30:00Z", "Thu 1 Jan at 12:30am"},
		// Cross-year.
		{"2025-12-25T18:00:00Z", "Dec 25 at 6:00pm 2025"},
		{"2024-03-15T09:00:00Z", "Mar 15 at 9:00am 2024"},
		// Day-only values (fall through to shortDay).
		{"2026-09-17T00:00:00Z", "Today"},
		{"2026-09-18T00:00:00Z", "Tomorrow"},
		{"2026-09-16T00:00:00Z", "Yesterday"},
		{"2026-09-14T00:00:00Z", "Mon 14 Sep"},
		{"2026-08-01T00:00:00Z", "1 Aug"},
		{"2025-03-15T00:00:00Z", "15 Mar 2025"},
		// Not a stored value — pass through unchanged.
		{"not a date", "not a date"},
		{"", ""},
	}

	for _, c := range cases {
		got := Relative(c.v, now)
		if got != c.want {
			t.Errorf("Relative(%q, %s) = %q, want %q", c.v, now.Format("2006-01-02"), got, c.want)
		}
	}
}
