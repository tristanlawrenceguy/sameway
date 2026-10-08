package server_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A late task is a press from done, tomorrow or today; several late are
// one press from today, and one Undo takes them all back.
func TestLateTasksAreAPressFromDealtWith(t *testing.T) {
	a, h := newApp(t)
	now := time.Now()
	day := func(n int) string { return now.AddDate(0, 0, n).Format("2006-01-02") }
	rent, _ := a.Store.Create("task", map[string]any{"title": "Pay rent", "due": day(-3)})
	passport, _ := a.Store.Create("task", map[string]any{"title": "Renew passport", "due": day(-1)})
	call, _ := a.Store.Create("task", map[string]any{"title": "Call the bank", "due": now.AddDate(0, 0, -2).Format("2006-01-02") + "T09:30:00Z"})
	page := get(t, h, "/today").Body.String()
	for _, want := range []string{"3 tasks are late", "Move all 3 to today", ">Done<", ">Tomorrow<"} {
		if !strings.Contains(page, want) {
			t.Fatalf("Today offers %q: %s", want, truncate(page))
		}
	}
	postForm(t, h, "/today/done", url.Values{"id": {rent.ID}})
	if got, _ := a.Store.Get("task", rent.ID); got.Fields["done"] != true {
		t.Error("Done marks it done")
	}
	postForm(t, h, "/today/move", url.Values{"id": {passport.ID}, "to": {"tomorrow"}})
	if got, _ := a.Store.Get("task", passport.ID); !strings.HasPrefix(got.Fields["due"].(string), day(1)) {
		t.Errorf("Tomorrow moves it a day past today: %v", got.Fields["due"])
	}
	a.Store.Update("task", passport.ID, map[string]any{"due": day(-1)})
	postForm(t, h, "/today/late", nil)
	for _, id := range []string{passport.ID, call.ID} {
		got, _ := a.Store.Get("task", id)
		if due, _ := got.Fields["due"].(string); !strings.HasPrefix(due, day(0)) {
			t.Errorf("all late are due today: %v", got.Fields)
		}
	}
	if got, _ := a.Store.Get("task", call.ID); !strings.Contains(got.Fields["due"].(string), "T") {
		t.Errorf("a task with a time keeps it: %v", got.Fields["due"])
	}
	entries, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err := a.Chat.UndoAs("human", entries[0].ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.Store.Get("task", passport.ID); !strings.HasPrefix(got.Fields["due"].(string), day(-1)) {
		t.Errorf("one Undo puts them all back: %v", got.Fields["due"])
	}
	if page := get(t, h, "/activity").Body.String(); !strings.Contains(page, "late tasks to today") {
		t.Errorf("the log says what moved: %s", truncate(page))
	}
}
