package track

import (
	"testing"
	"time"
)

func at(day string, hour int) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", day, time.Local)
	return t.Add(time.Duration(hour) * time.Hour)
}

// A daily habit done once a day: the streak counts back from today when
// today is done, and from yesterday when it is not, since a day still
// going is not a day missed; the best run is the longest there has been.
func TestADailyStreak(t *testing.T) {
	h := Habit{Name: "Stretch", Cadence: "day"}
	entries := []Entry{
		{at("2026-09-15", 8), 1}, {at("2026-09-16", 8), 1}, {at("2026-09-17", 8), 1}, {at("2026-09-18", 8), 1},
		{at("2026-09-20", 8), 1}, {at("2026-09-21", 8), 1},
	}
	now := at("2026-09-22", 10)
	s := Summarise(h, entries, now, 7)
	if s.Met || s.Now != 0 || s.Streak != 2 || s.Best != 4 {
		t.Errorf("today not yet done: streak 2 from yesterday, best 4, got %+v", s)
	}
	if len(s.Last) != 7 || !s.Last[6].Start.Equal(at("2026-09-22", 0)) || s.Last[5].Met != true || s.Last[4].Met != true || s.Last[3].Met != false {
		t.Errorf("the last seven days, oldest first, today last: %+v", s.Last)
	}
	if Progress(h, s) != "not yet" {
		t.Errorf("a once-a-day habit not done says so, got %q", Progress(h, s))
	}
	s = Summarise(h, append(entries, Entry{now, 1}), now, 7)
	if !s.Met || s.Streak != 3 || Progress(h, s) != "done" {
		t.Errorf("done today: streak 3, got %+v %q", s, Progress(h, s))
	}
}

// A measured habit with a target and a unit: the period adds up, the
// target decides met, and a goal shows how far it has come.
func TestAMeasuredHabitWithATargetAndAGoal(t *testing.T) {
	h := Habit{Name: "Run", Cadence: "week", Target: 15, Unit: "km", Goal: 100}
	entries := []Entry{
		{at("2026-09-08", 7), 5}, {at("2026-09-10", 7), 5}, {at("2026-09-12", 7), 6},
		{at("2026-09-15", 7), 8}, {at("2026-09-17", 7), 4},
		{at("2026-09-21", 7), 10},
	}
	now := at("2026-09-22", 12)
	s := Summarise(h, entries, now, 4)
	if s.Now != 10 || s.Met || s.Total != 38 || s.Pct != 38 {
		t.Errorf("this week 10 of 15 km, 38 of the goal: %+v", s)
	}
	if s.Streak != 0 || s.Best != 1 {
		t.Errorf("last week missed the target so the streak is 0 from it, the week before met it: %+v", s)
	}
	if Progress(h, s) != "10 of 15 km" || Amount(2.5, "km") != "2.5 km" || Amount(8, "glasses") != "8 glasses" || Amount(3, "") != "3" {
		t.Errorf("amounts read as a person would: %q %q", Progress(h, s), Amount(2.5, "km"))
	}
	if StreakWords(1, h) != "1 week in a row" || StreakWords(3, Habit{}) != "3 days in a row" {
		t.Error("streak words")
	}
	if PeriodStart(at("2026-09-22", 12), "week") != at("2026-09-21", 0) {
		t.Errorf("a week starts on its Monday, got %v", PeriodStart(at("2026-09-22", 12), "week"))
	}
}
