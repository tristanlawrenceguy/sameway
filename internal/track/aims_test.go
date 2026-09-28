package track

import "testing"

// A monthly allowance: hours worked add up over the month, the limit is
// kept while they stay under it, and what goes past it is the over.
func TestAMonthlyLimit(t *testing.T) {
	h := Habit{Name: "Hours", Cadence: "month", Target: 21, Aim: Limit, Unit: "hours"}
	entries := []Entry{
		{at("2026-07-06", 9), 2}, {at("2026-07-09", 9), 2},
		{at("2026-08-03", 9), 12}, {at("2026-08-20", 9), 12},
		{at("2026-09-01", 9), 2}, {at("2026-09-03", 9), 3.5}, {at("2026-09-17", 9), 4},
	}
	now := at("2026-09-23", 12)
	s := Summarise(h, entries, now, 4)
	if s.Now != 9.5 || !s.Met || s.Left != 11.5 || s.Over != 0 {
		t.Errorf("9.5 of 21 hours, within, 11.5 left: %+v", s)
	}
	if got := Progress(h, s); got != "9.5 of 21 hours, 11.5 left" {
		t.Errorf("progress for a limit says what is left, got %q", got)
	}
	if s.Streak != 1 || s.Best != 1 {
		t.Errorf("August went over, so the run is this month alone; July before it was within: %+v", s)
	}
	if len(s.Last) != 4 || s.Last[0].Met || !s.Last[1].Met || s.Last[2].Met || s.Last[2].Amount != 24 {
		t.Errorf("June is before anything was tracked so it does not count, July within, August 24 hours: %+v", s.Last)
	}
	s = Summarise(h, append(entries, Entry{now, 15}), now, 4)
	if s.Met || s.Over != 3.5 || Progress(h, s) != "24.5 of 21 hours, 3.5 over" {
		t.Errorf("past the limit says how far: %+v %q", s, Progress(h, s))
	}
	if StreakWords(2, h) != "2 months in a row within the limit" {
		t.Errorf("streak words for a limit, got %q", StreakWords(2, h))
	}
	if PeriodStart(at("2026-09-23", 12), "month") != at("2026-09-01", 0) || PeriodLabel(at("2026-09-01", 0), "month") != "2026-09" {
		t.Error("a month starts on its first and is labelled by its month")
	}
}

// A measure that is only recorded, where a period is its latest reading:
// no target, no streak, and a period with nothing in it says so.
func TestARecordOfTheLatest(t *testing.T) {
	h := Habit{Name: "Weight", Aim: Record, Combine: Latest, Unit: "kg"}
	entries := []Entry{{at("2026-09-21", 7), 73.1}, {at("2026-09-22", 7), 72.8}, {at("2026-09-22", 21), 73.4}}
	s := Summarise(h, entries, at("2026-09-22", 22), 3)
	if s.Now != 73.4 || s.Met || s.Streak != 0 || s.Best != 0 || s.Target != 0 || Progress(h, s) != "73.4 kg" {
		t.Errorf("the latest of the day, no target and no streak: %+v %q", s, Progress(h, s))
	}
	if s.Last[0].Logged || !s.Last[1].Logged || s.Last[1].Amount != 73.1 {
		t.Errorf("a day with nothing logged is not a reading of nought: %+v", s.Last)
	}
	s = Summarise(h, entries, at("2026-09-23", 8), 3)
	if s.Logged || Progress(h, s) != "nothing yet" {
		t.Errorf("nothing today yet, got %q", Progress(h, s))
	}
}

// An average to reach over a week, and a yearly count.
func TestAnAverageAndAYear(t *testing.T) {
	sleep := Habit{Name: "Sleep", Cadence: "week", Target: 7, Aim: Reach, Combine: Average, Unit: "hours"}
	s := Summarise(sleep, []Entry{{at("2026-09-21", 7), 6}, {at("2026-09-22", 7), 8}, {at("2026-09-23", 7), 7.6}}, at("2026-09-23", 12), 2)
	if s.Now != 7.2 || !s.Met || s.Total != 21.6 {
		t.Errorf("an average of 7.2 hours meets 7: %+v", s)
	}
	books := Habit{Name: "Books", Cadence: "year", Target: 20}
	s = Summarise(books, []Entry{{at("2025-03-01", 9), 1}, {at("2026-02-01", 9), 1}, {at("2026-06-01", 9), 1}}, at("2026-09-23", 12), 3)
	if s.Now != 2 || s.Met || Progress(books, s) != "2 of 20" || PeriodLabel(s.Last[2].Start, "year") != "2026" || s.Last[1].Amount != 1 {
		t.Errorf("two books this year of twenty: %+v", s)
	}
}
