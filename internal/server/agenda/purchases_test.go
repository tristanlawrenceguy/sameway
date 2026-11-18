package agenda_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What was bought is said three days before the last day to return it and
// a month before its warranty ends, once each, and not once returned.
func TestReturnsAndWarrantiesAreSaidAhead(t *testing.T) {
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
	phones, _ := a.Store.Create("purchase", map[string]any{"item": "Headphones", "return_by": day(2), "warranty_until": day(400)})
	a.Store.Create("purchase", map[string]any{"item": "Dishwasher", "warranty_until": day(20)})
	a.Store.Create("purchase", map[string]any{"item": "Kettle", "return_by": day(1), "returned": true})
	a.Store.Create("purchase", map[string]any{"item": "Lamp", "return_by": day(-2)})
	if n := h.PurchasesIfDue(now); n != 2 {
		t.Fatalf("the return near and the warranty within a month, got %d", n)
	}
	if n := h.PurchasesIfDue(now.Add(time.Hour)); n != 0 {
		t.Errorf("each is said once, got %d more", n)
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	all := strings.Join(sent, "\n")
	if !strings.Contains(all, "The last day to return Headphones is ") || strings.Contains(all, "is on ") || !strings.Contains(all, "/t/purchase/"+phones.ID) {
		t.Errorf("the return is said, leading to its page: %v", sent)
	}
	if !strings.Contains(all, "The warranty on Dishwasher ends on ") || strings.Contains(all, "Kettle") || strings.Contains(all, "Lamp") {
		t.Errorf("the warranty is said; what is returned or gone by is not: %v", sent)
	}
}
