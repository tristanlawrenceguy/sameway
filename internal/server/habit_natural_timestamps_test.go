package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestHabitDetailLedeNoMachineFormatDate asserts that a habit detail page's
// lede uses natural language for timestamps — no full machine-format date
// like "Wed 30 Sep 2026, 13:31" (acceptance item 1). The whenMade function
// should produce relative phrasing such as "4 days ago at 1:31pm".
func TestHabitDetailLedeNoMachineFormatDate(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("habit", map[string]any{
		"name":   "Water",
		"target": 8.0,
		"unit":   "glasses",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/habit/"+rec.ID+fieldsView).Body.String()

	// Extract the sw-detail__when span content.
	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("habit detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must not contain a machine-format date with a full year.
	machineYearRe := regexp.MustCompile(`\d{4}`)
	if machineYearRe.MatchString(whenText) {
		t.Errorf("habit detail lede should use natural language timestamps, got %q — contains a 4-digit year (machine format like Wed 30 Sep 2026)", whenText)
	}

	// The when text must not contain a 24-hour clock pattern ", HH:MM".
	machineTimeRe := regexp.MustCompile(`,\s\d{1,2}:\d{2}\b`)
	if machineTimeRe.MatchString(whenText) {
		t.Errorf("habit detail lede should use natural language timestamps, got %q — contains a 24-hour clock pattern (machine format like Wed 30 Sep 2026, 13:31)", whenText)
	}

	// It must still say "Started" before the date.
	if !strings.HasPrefix(whenText, "Started ") {
		t.Errorf("habit detail lede should start with 'Started ', got %q", whenText)
	}
}
