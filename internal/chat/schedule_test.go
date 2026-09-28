package chat_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An action runs on its own at the times a person gave it, once per period,
// and remembers when, so a restart does not run it again.
func TestActionsRunOnTheirOwnAtTheirTime(t *testing.T) {
	at := func(every, at, on, last string) *store.Record {
		f := map[string]any{"every": every, "at": at, "on": on}
		if last != "" {
			f["last_run"] = last
		}
		return &store.Record{Fields: f}
	}
	tuesday := time.Date(2026, 9, 15, 7, 30, 0, 0, time.UTC) // a Tuesday
	cases := []struct {
		name string
		rec  *store.Record
		now  time.Time
		want bool
	}{
		{"never", at("never", "", "", ""), tuesday, false},
		{"hourly, never ran", at("hour", "", "", ""), tuesday, true},
		{"hourly, ran 20 minutes ago", at("hour", "", "", "2026-09-15T07:10:00Z"), tuesday, false},
		{"hourly, ran 2 hours ago", at("hour", "", "", "2026-09-15T05:10:00Z"), tuesday, true},
		{"daily at 07:00, now 07:30, not yet today", at("day", "07:00", "", "2026-09-14T07:01:00Z"), tuesday, true},
		{"daily at 07:00, already ran today", at("day", "07:00", "", "2026-09-15T07:01:00Z"), tuesday, false},
		{"daily at 08:00, too early", at("day", "08:00", "", ""), tuesday, false},
		{"weekly on tuesday at 07:00", at("week", "07:00", "tuesday", ""), tuesday, true},
		{"weekly on monday", at("week", "07:00", "monday", ""), tuesday, false},
	}
	for _, c := range cases {
		if got := chat.Due(c.rec, c.now); got != c.want {
			t.Errorf("%s: due=%v, want %v", c.name, got, c.want)
		}
	}

	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		io.WriteString(w, "sunny")
	}))
	defer remote.Close()
	chat.HTTPClient = remote.Client()
	svc := newFullService(t)
	weather, _ := svc.Store.Create(chat.ActionType, map[string]any{"title": "Weather", "url": remote.URL, "every": "hour", "show": true})
	svc.Store.Create(chat.ActionType, map[string]any{"title": "Alarm", "url": remote.URL, "every": "never"})

	ran := svc.RunDue(context.Background(), tuesday)
	if len(ran) != 1 || ran[0] != weather.ID || calls != 1 {
		t.Fatalf("only the hourly action is due, got %v after %d calls", ran, calls)
	}
	if again := svc.RunDue(context.Background(), tuesday.Add(5*time.Minute)); len(again) != 0 {
		t.Error("five minutes later nothing is due")
	}
	if later := svc.RunDue(context.Background(), tuesday.Add(61*time.Minute)); len(later) != 1 || calls != 2 {
		t.Errorf("an hour later it runs again, got %v after %d calls", later, calls)
	}
	rec, _ := svc.Store.Get(chat.ActionType, weather.ID)
	if rec.Fields["last_run"] == "" {
		t.Error("the action should remember when it ran")
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 1 || blocks[0].Fields["props"].(map[string]any)["content"] != "sunny" {
		t.Errorf("a scheduled webhook with show keeps one block current, got %v", blocks)
	}
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{})
	system := 0
	for _, e := range log {
		if e.Fields["actor"] == "system" && e.Fields["action"] == "ran" {
			system++
		}
	}
	if system != 2 {
		t.Errorf("scheduled runs are logged as the system's, got %d", system)
	}
}
