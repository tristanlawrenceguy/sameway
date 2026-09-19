package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The clock sets a timer or an alarm as a reminder record, lists what is
// coming with what is on today, rings a reminder when its time comes
// over the stream an open page listens on, and lets the person dismiss it
// or have five more minutes.
func TestTheClockSetsListsAndRings(t *testing.T) {
	a, h := newApp(t)
	blk, err := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "clock", "props": map[string]any{}}))
	if err != nil {
		t.Fatal(err)
	}
	// A task due today at four sits under the time, from the calendar's types.
	today := time.Now()
	four := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.Local)
	a.Store.Create("task", map[string]any{"title": "Water the tomatoes", "due": when.Store(four, false)})

	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, `data-component="clock"`) || !strings.Contains(page, "Water the tomatoes") || !strings.Contains(page, `class="sw-clock__when">16:00<`) {
		t.Errorf("the clock is on the canvas with what is on today\n%s", page)
	}

	// A timer, from the form: a reminder ten minutes from now.
	req := httptest.NewRequest(http.MethodPost, "/clock/set", strings.NewReader(url.Values{"minutes": {"10"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/canvas/"+blk.ID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/canvas/"+blk.ID {
		t.Errorf("setting a timer goes back to the page it came from, got %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	reminders, _ := a.Store.List(server.ReminderType, store.ListOptions{})
	if len(reminders) != 1 || reminders[0].Fields["kind"] != "timer" || reminders[0].Fields["title"] != "10 minute timer" {
		t.Fatalf("a timer is a reminder record, got %v", reminders)
	}
	at, _ := time.Parse(time.RFC3339, reminders[0].Fields["at"].(string))
	if d := time.Until(at); d < 9*time.Minute || d > 11*time.Minute {
		t.Errorf("the timer ends ten minutes from now, got %v", d)
	}
	page = get(t, h, "/").Body.String()
	if !strings.Contains(page, `data-kind="timer"`) || !strings.Contains(page, "10 minute timer") {
		t.Errorf("the timer is listed as coming up\n%s", page)
	}

	// An alarm for a time of day, named for what it is for.
	wantStatus(t, postForm(t, h, "/clock/set", url.Values{"at": {"07:30"}, "title": {"Call the vet"}}), http.StatusSeeOther)
	reminders, _ = a.Store.List(server.ReminderType, store.ListOptions{})
	if len(reminders) != 2 {
		t.Fatalf("an alarm is a reminder too, got %d", len(reminders))
	}

	// A reminder whose time has come rings on the stream, once, and is
	// marked rung; the page then shows it ringing until it is dismissed.
	due, _ := a.Store.Create(server.ReminderType, map[string]any{"title": "Tea", "at": when.Store(time.Now().Add(-time.Minute), false)})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sreq := httptest.NewRequest(http.MethodGet, "/clock/stream", nil).WithContext(ctx)
	srec := httptest.NewRecorder()
	h.ServeHTTP(srec, sreq)
	if out := srec.Body.String(); !strings.Contains(out, "event: ring") || !strings.Contains(out, `"title":"Tea"`) || strings.Count(out, "event: ring") != 1 {
		t.Errorf("the due reminder rings once on the stream\n%s", out)
	}
	rung, _ := a.Store.Get(server.ReminderType, due.ID)
	if rung.Fields["state"] != "rang" {
		t.Errorf("a rung reminder is marked so, got %v", rung.Fields["state"])
	}
	page = get(t, h, "/").Body.String()
	if !strings.Contains(page, `class="sw-clock__ring" data-id="`+due.ID+`"`) || !strings.Contains(page, "/clock/"+due.ID+"/done") {
		t.Errorf("a rung reminder shows ringing with Dismiss\n%s", page)
	}
	if !strings.Contains(get(t, h, "/activity").Body.String(), "rang") {
		t.Error("the ringing is in the activity log")
	}

	// Five more minutes sets it again, later; Dismiss is done.
	wantStatus(t, postForm(t, h, "/clock/"+due.ID+"/snooze", nil), http.StatusSeeOther)
	snoozed, _ := a.Store.Get(server.ReminderType, due.ID)
	later, _ := time.Parse(time.RFC3339, snoozed.Fields["at"].(string))
	if snoozed.Fields["state"] != "set" || time.Until(later) < 4*time.Minute {
		t.Errorf("five more minutes sets it again, later, got %v at %v", snoozed.Fields["state"], later)
	}
	wantStatus(t, postForm(t, h, "/clock/"+due.ID+"/done", nil), http.StatusSeeOther)
	done, _ := a.Store.Get(server.ReminderType, due.ID)
	if done.Fields["state"] != "done" {
		t.Errorf("dismissed is done, got %v", done.Fields["state"])
	}
}
