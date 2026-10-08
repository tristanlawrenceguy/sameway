package blocks

import "time"

// Which month or day a calendar shows, and the way to those either side.

// shownMonth is the month a calendar shows, this one unless it says
// another, and today, filled in out; and the one day it shows instead,
// when it does, whose month is then the one shown.
func shownMonth(out map[string]any, now time.Time) (month, day string, shownDay time.Time) {
	month, _ = out["month"].(string)
	if _, err := time.Parse("2006-01", month); err != nil {
		month = now.Format("2006-01")
		out["month"] = month
	}
	if d, _ := out["today"].(string); d == "" {
		out["today"] = now.Format("2006-01-02")
	}
	// One day instead of the month: its month is the one shown, and the
	// days either side are a link away, as the months are.
	day, _ = out["day"].(string)
	shownDay, dayErr := time.Parse("2006-01-02", day)
	if day != "" && dayErr != nil {
		delete(out, "day")
		day = ""
	}
	if day != "" {
		out["month"] = shownDay.Format("2006-01")
		month = out["month"].(string)
	}
	return month, day, shownDay
}

// calendarNav is the way to the months, or the days, either side, on the
// block's own page.
func calendarNav(out map[string]any, blockID, month, day string, shownDay, now time.Time) {
	base := "/canvas/" + blockID
	out["dayBase"] = base + "?day="
	if day != "" {
		prev, next := shownDay.AddDate(0, 0, -1), shownDay.AddDate(0, 0, 1)
		out["nav"] = map[string]any{
			"previous": map[string]any{"href": base + "?day=" + prev.Format("2006-01-02"), "label": prev.Format("Mon 2 Jan")},
			"next":     map[string]any{"href": base + "?day=" + next.Format("2006-01-02"), "label": next.Format("Mon 2 Jan")},
			"month":    map[string]any{"href": base + "?month=" + month, "label": shownDay.Format("January")},
		}
		return
	}
	shown, _ := time.Parse("2006-01", month)
	prev, next := shown.AddDate(0, -1, 0), shown.AddDate(0, 1, 0)
	out["nav"] = map[string]any{
		"previous": map[string]any{"href": base + "?month=" + prev.Format("2006-01"), "label": prev.Format("January 2006")},
		"next":     map[string]any{"href": base + "?month=" + next.Format("2006-01"), "label": next.Format("January 2006")},
		"today":    map[string]any{"href": base + "?day=" + now.Format("2006-01-02"), "label": "Today"},
	}
}
