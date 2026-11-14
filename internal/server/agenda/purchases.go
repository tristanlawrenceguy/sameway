package agenda

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// What was bought once is said before its last day to return it, three
// days ahead, and before its warranty ends, a month ahead, once each,
// while there is still time to act; nothing more once it is returned.

const purchaseType = "purchase"

// lastDays are the days of a purchase worth saying ahead: how long ahead,
// and the words.
var lastDays = []struct {
	field string
	ahead time.Duration
	words func(item, on string) string
}{
	{"return_by", 3 * 24 * time.Hour, func(item, on string) string {
		return "You can still return " + item + " until " + strings.TrimPrefix(on, "on ") + "."
	}},
	{"warranty_until", 30 * 24 * time.Hour, func(item, on string) string {
		return "The warranty on " + item + " ends " + on + ". If anything is wrong with it, claim before then."
	}},
}

// PurchasesIfDue says each last day to return something, and each
// warranty's end, that comes within its time and has not been said, and
// says how many it said.
func (s *Service) PurchasesIfDue(now time.Time) int {
	if _, ok := s.app.Types.Get(purchaseType); !ok || !s.CanNotify() {
		return 0
	}
	recs, _ := s.app.Store.List(purchaseType, store.ListOptions{})
	said := 0
	for _, p := range recs {
		if back, _ := p.Fields["returned"].(bool); back {
			continue
		}
		for _, d := range lastDays {
			if s.lastDaySaid(p, d.field, d.ahead, now, d.words) {
				said++
			}
		}
	}
	return said
}

// lastDaySaid says one of a purchase's last days when it is near and not
// over, once, and says whether it did.
func (s *Service) lastDaySaid(p *store.Record, field string, ahead time.Duration, now time.Time, words func(item, on string) string) bool {
	v, _ := p.Fields[field].(string)
	at, _, ok := when.Stored(v, now)
	if !ok {
		return false
	}
	day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, now.Location())
	key := field + ":" + p.ID + ":" + v[:min(len(v), 10)]
	if !day.AddDate(0, 0, 1).After(now) || day.Sub(now) > ahead || s.app.Store.Meta(key) != "" {
		return false
	}
	s.app.Store.SetMeta(key, "said")
	item := strings.TrimSpace(fmt.Sprint(p.Fields["item"]))
	title := "Last day to return it soon"
	if field == "warranty_until" {
		title = "Warranty ends soon"
	}
	s.Notify(title, words(item, whenWords(day, now)), relate.Page(purchaseType, p.ID))
	return true
}
