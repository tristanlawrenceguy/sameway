package agenda

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// What is paid again and again is said three days before it renews, once
// for each renewal, while there is still time to cancel; a renewal gone by
// moves on to the next by its repeat.

const (
	subscriptionType = "subscription"
	renewalNotice    = 3 * 24 * time.Hour
)

// RenewalsIfDue moves on each renewal gone by, a free trial then paid, and says each that comes
// within three days and has not been said, and says how many it said.
func (s *Service) RenewalsIfDue(now time.Time) int {
	if _, ok := s.app.Types.Get(subscriptionType); !ok {
		return 0
	}
	subs, _ := s.app.Store.List(subscriptionType, store.ListOptions{})
	said := 0
	for _, sub := range subs {
		v, _ := sub.Fields["renews"].(string)
		at, _, ok := when.Stored(v, now)
		if !ok {
			continue
		}
		day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, now.Location())
		if !day.AddDate(0, 0, 1).After(now) { // its day is over
			if next, ok := when.Next(fmt.Sprint(sub.Fields["repeat"]), v, now); ok && next != v {
				records.WriteAs(s.app.Store, records.Who{Actor: "system", Via: "a renewal gone by"}, "updated", subscriptionType, sub.ID, map[string]any{"renews": next, "trial": false}) // a trial over is paid for
			}
			continue
		}
		key := "renewal:" + sub.ID + ":" + v[:min(len(v), 10)]
		if day.Sub(now) > renewalNotice || !s.CanNotify() || s.app.Store.Meta(key) != "" {
			continue
		}
		s.app.Store.SetMeta(key, "said")
		s.Notify(renewalWords(sub, day, now))
		said++
	}
	return said
}

// renewalWords are the notice: what renews, when, for how much, and that
// it can still be cancelled, leading to its page and the link there.
func renewalWords(sub *store.Record, day, now time.Time) (title, text, path string) {
	name := strings.TrimSpace(fmt.Sprint(sub.Fields["name"]))
	price, _ := sub.Fields["price"].(string)
	on := whenWords(day, now)
	if trial, _ := sub.Fields["trial"].(bool); trial {
		text = name + "'s free trial ends " + on
		if price != "" {
			text += ", then it costs " + price
		}
	} else {
		text = name + " renews " + on
		if price != "" {
			text += " for " + price
		}
	}
	return "Renews soon", text + ". To stop it, cancel before then.", relate.Page(subscriptionType, sub.ID)
}

// whenWords is the day as a person says it ahead: today, tomorrow, on
// Friday.
func whenWords(day, now time.Time) string {
	w := when.Day(day, now)
	if w == "Today" || w == "Tomorrow" {
		return strings.ToLower(w)
	}
	return "on " + w
}
