package render

import (
	"strings"
	"testing"
)

// A value below zero is a bar going down from the zero line, not an empty
// bar at the foot with its number over it.
func TestANegativeValueGoesDownFromZero(t *testing.T) {
	s := chartShape([]any{map[string]any{"label": "Aug", "value": 40.0}, map[string]any{"label": "Sep", "value": -20.0}}, "bar")
	up, down := s.Points[0], s.Points[1]
	if down.H <= 0 || down.Y != s.Baseline || up.Y+up.H != s.Baseline {
		t.Errorf("both bars should meet the zero line, one up, one down: %+v %+v zero %v", up, down, s.Baseline)
	}
	if s.Bottom <= s.Baseline {
		t.Errorf("the plot should reach below zero for a value under it")
	}
	if down.LabelY <= down.Y+down.H {
		t.Errorf("a value going down is written under its bar")
	}
}

func TestNumbersAreWrittenAsAPersonWritesThem(t *testing.T) {
	for v, want := range map[float64]string{12500: "12,500", 1500: "1,500", 999: "999", 295.5: "295.5", -1250: "-1,250", 1234567.25: "1,234,567.25"} {
		if got := numberText(v); got != want {
			t.Errorf("numberText(%v) = %q, want %q", v, got, want)
		}
	}
}

// A chart given no description of its own says what it shows in words.
func TestAChartSaysWhatItShows(t *testing.T) {
	series := []any{map[string]any{"label": "2026-09-14", "value": 8.0}, map[string]any{"label": "2026-09-15", "value": 5.0}, map[string]any{"label": "2026-09-16", "value": 9.0}}
	got := chartSummary(series, "glasses", 8.0, "target")
	for _, want := range []string{"From 8 glasses for 14 Sep to 9 glasses for 16 Sep", "Highest 9 glasses, for 16 Sep", "lowest 5 glasses, for 15 Sep", "Reached the target of 8 glasses for 2 of 3"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary %q should say %q", got, want)
		}
	}
	if got := chartSummary(series, "", 6.0, "limit"); !strings.Contains(got, "Within the limit of 6 for 1 of 3") {
		t.Errorf("a limit summary says how often it was kept within: %q", got)
	}
}
