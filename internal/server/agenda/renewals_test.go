package agenda_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What is paid again and again is said three days before it renews, once,
// and a renewal gone by moves on by its repeat, a free trial then paid.
func TestRenewalsAreSaidAhead(t *testing.T) {
	t.Parallel()
	a, _ := newAppAt(t, tonight)
	h := server.New(a)
	now := tonight
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") + "T00:00:00Z" }
	var mu sync.Mutex
	var sent []string
	h.OnRing(func(title, text, link string) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, title+" | "+text+" | "+link)
	})
	soon, _ := a.Store.Create("subscription", map[string]any{"name": "Streamflix", "price": "$15.99", "repeat": "every month", "renews": day(2)})
	a.Store.Create("subscription", map[string]any{"name": "Later Music", "price": "$9.99", "renews": day(20)})
	gone, _ := a.Store.Create("subscription", map[string]any{"name": "Photo Cloud", "repeat": "every month", "renews": day(-1), "trial": true})
	if n := h.RenewalsIfDue(now); n != 1 {
		t.Fatalf("only what renews within three days is said, got %d", n)
	}
	if n := h.RenewalsIfDue(now.Add(time.Hour)); n != 0 {
		t.Errorf("a renewal is said once, got %d more", n)
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if len(sent) != 1 || !strings.Contains(sent[0], "Streamflix renews on ") || !strings.Contains(sent[0], "for $15.99. To stop it, cancel before then.") || !strings.HasSuffix(sent[0], "/t/subscription/"+soon.ID) {
		t.Errorf("it says what renews, when and for how much, leading to its page: %v", sent)
	}
	mu.Unlock()
	moved, _ := a.Store.Get("subscription", gone.ID)
	if r, _ := moved.Fields["renews"].(string); !strings.HasPrefix(r, now.AddDate(0, 1, -1).Format("2006-01-02")) || moved.Fields["trial"] != false {
		t.Errorf("a renewal gone by moves on a month and the trial is over: %v", moved.Fields)
	}
}
