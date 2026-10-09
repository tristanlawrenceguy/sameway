package server_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A long Today starts with three, what matters first, the rest folded
// away; a short one is shown whole. Everything in the person's head, put
// in the box, goes to the assistant to sort.
func TestALongTodayStartsWithThree(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := time.Now()
	day := func(n int) string { return now.AddDate(0, 0, n).Format("2006-01-02") }
	for i, title := range []string{"Renew passport", "Pay rent", "Call the bank", "Book dentist"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-1 - i)})
	}
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Start with these") || !strings.Contains(page, "Too much in your head") {
		t.Fatalf("four is read whole, with the box: %s", truncate(page))
	}
	a.Store.Create("task", map[string]any{"title": "Reply to the school", "due": day(0)})
	a.Store.Create("task", map[string]any{"title": "Send the tax form", "due": day(0), "tags": []any{"important"}})
	page := get(t, h, "/today").Body.String()
	start := strings.Index(page, "Start with these")
	rest := strings.Index(page, "The rest")
	if start < 0 || rest < start {
		t.Fatalf("six start with three, the rest after: %s", truncate(page))
	}
	first := page[start:rest]
	for _, want := range []string{"Send the tax form", "Book dentist", "Call the bank"} {
		if !strings.Contains(first, want) {
			t.Errorf("what matters, then the latest late, start: %q missing from %s", want, first)
		}
	}
	if strings.Index(first, "Send the tax form") > strings.Index(first, "Book dentist") {
		t.Error("what matters comes first")
	}
	if !strings.Contains(page[rest:], "Renew passport") {
		t.Error("the rest are there, folded")
	}

	postForm(t, h, "/today/sort", url.Values{"words": {"dentist friday, call mum, taxes due 31st"}})
	msgs, _ := a.Store.List("message", store.ListOptions{})
	asked := false
	for _, m := range msgs {
		asked = asked || m.Fields["role"] == "user" && strings.Contains(m.Fields["content"].(string), "call mum") && strings.Contains(m.Fields["content"].(string), "three to start with")
	}
	if !asked {
		t.Error("the box goes to the assistant, asked to sort it")
	}
}
