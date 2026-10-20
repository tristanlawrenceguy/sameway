package server_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Every time a page shows is said with what it is, and held in a <time>
// for a machine (design/foundations/glance.md, "Said with what it is").

// A day field called at is named by its label, from the schema, on every
// type: an entry is Logged, an interaction Happened, a reminder Rings.
func TestADayIsNamedByItsLabel(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	habit, err := a.Store.Create("habit", map[string]any{"name": "Water", "target": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := a.Store.Create("entry", map[string]any{"habit": habit.ID, "at": tonight.Add(-2 * 3600e9).UTC().Format("2006-01-02T15:04:05Z")})
	if err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/t/entry/"+entry.ID).Body.String(); !strings.Contains(page, ">Logged today at ") {
		t.Error("an entry's day is not named Logged on its page")
	}
	// The line under its fields names when it was made, lower case after
	// its word: created today at, never created Today at. No label text — just the time element.
	made := regexp.MustCompile(`sw-detail__when[^"]*"><time[^>]*>today at `)
	if page := get(t, h, "/t/entry/"+entry.ID).Body.String(); !made.MatchString(page) {
		t.Error("when it was made is not said as Created today at")
	}
}

// The days over the log, a chat in the chat menu and a task's day on the
// weekly review are each held in a <time> with its date.
func TestDaysOverListsAreHeldInTimeElements(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	day := tonight.Format("2006-01-02")
	postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Buy paint", "due": tonight.AddDate(0, 0, 1).Format("2006-01-02")})
	if page := get(t, h, "/activity").Body.String(); !strings.Contains(page, `id="day-`+day+`"><time datetime="`+day+`">Today, `) {
		t.Error("a day over the log is not held in a time element")
	}
	if page := get(t, h, "/review").Body.String(); !strings.Contains(page, ` <span class="sw-row__meta"><time datetime="`+tonight.AddDate(0, 0, 1).Format("2006-01-02")+`">Tomorrow</time>`) {
		t.Error("a task's day on the weekly review is not held in a time element")
	}
	if page := get(t, h, "/chat").Body.String(); strings.Contains(page, `class="sw-chat__open`) && !strings.Contains(page, `<time class="sw-muted sw-small" datetime="`) {
		t.Error("a chat's day in the chat menu is not held in a time element")
	}
}
