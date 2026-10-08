package server_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// The weekly review shows what was done, what slipped (each moved to next
// week or ticked in one press) and the week ahead; on the day set, in the
// evening, it says once that it is ready.
func TestTheWeeklyReview(t *testing.T) {
	a, _ := newApp(t)
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") }
	ids := map[string]string{}
	for _, f := range []map[string]any{
		{"title": "Pay the plumber", "due": day(-1), "done": true},
		{"title": "Renew passport", "due": day(-3)},
		{"title": "Call the bank", "due": day(-2)},
		{"title": "Send the invoice", "due": day(2)},
		{"title": "Plan the holiday", "due": day(30)},
	} {
		rec, err := a.Store.Create("task", f)
		if err != nil {
			t.Fatal(err)
		}
		ids[f["title"].(string)] = rec.ID
	}
	h := server.New(a)
	page := get(t, h, "/review").Body.String()
	for _, want := range []string{"Done this week", "Pay the plumber", "Slipped", "Renew passport", "The week ahead", "Send the invoice", "Plan next week with the assistant"} {
		if !strings.Contains(page, want) {
			t.Errorf("the review has %s", want)
		}
	}
	if strings.Contains(page, "Plan the holiday") {
		t.Error("next month is not the week ahead")
	}
	postForm(t, h, "/review/next", url.Values{"id": {ids["Renew passport"]}})
	postForm(t, h, "/review/done", url.Values{"id": {ids["Call the bank"]}})
	moved, _ := a.Store.Get("task", ids["Renew passport"])
	if !strings.HasPrefix(moved.Fields["due"].(string), day(7)) {
		t.Errorf("moved to next week: %v", moved.Fields["due"])
	}
	if bank, _ := a.Store.Get("task", ids["Call the bank"]); bank.Fields["done"] != true {
		t.Error("ticked")
	}
	var sent []string
	h.OnRing(func(title, text, link string) { sent = append(sent, title+" | "+link) })
	postForm(t, h, "/review/on", url.Values{"on": {strings.ToLower(now.Weekday().String())}})
	evening := time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.Local)
	if !h.ReviewIfDue(evening) || h.ReviewIfDue(evening) {
		t.Fatal("ready once on the day, in the evening")
	}
	time.Sleep(50 * time.Millisecond)
	if len(sent) != 1 || !strings.HasSuffix(sent[0], "/review") {
		t.Errorf("it says so, leading to the review: %v", sent)
	}
}
