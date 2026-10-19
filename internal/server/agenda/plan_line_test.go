package agenda_test

import (
	"strings"
	"testing"
	"time"
)

// On the day to plan the week (Sunday when no review day is chosen),
// Today points at the weekly review; on another day it says nothing.
func TestTodayPointsAtTheReviewOnItsDay(t *testing.T) {
	t.Parallel()
	_, sunday := newAppAt(t, time.Date(2026, 10, 18, 19, 0, 0, 0, time.UTC))
	if page := get(t, sunday, "/today").Body.String(); !strings.Contains(page, `Plan the week: <a class="sw-link" href="/review">Weekly review</a>`) {
		t.Errorf("on Sunday, Today points at the review: %s", truncate(page))
	}
	_, tuesday := newAppAt(t, time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC))
	if page := get(t, tuesday, "/today").Body.String(); strings.Contains(page, "Plan the week") {
		t.Error("on another day, nothing")
	}
}

// On a long Today, any day, the review is pointed at too: what needs
// attention this week, apart from the old overdue.
func TestALongTodayPointsAtTheReview(t *testing.T) {
	t.Parallel()
	tuesday := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	a, h := newAppAt(t, tuesday)
	for i := 1; i <= 9; i++ {
		a.Store.Create("task", map[string]any{"title": "Box " + string(rune('A'+i)), "due": tuesday.AddDate(0, 0, -i).Format("2006-01-02")})
	}
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Plan the week:") {
		t.Errorf("a long Today points at the review: %s", truncate(page))
	}
}
