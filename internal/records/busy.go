package records

import (
	"sort"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// When the person is busy, worked out from their events and nothing
// else: the times, never what they are. It is what another person's
// Sameway is told when it asks when they are free (chat/enquiry.go), so
// it is made by this code alone, from stored times.

// Span is a stretch of time.
type Span struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// busyMost is how many times one repeating event is counted in a stretch.
const busyMost = 200

// Busy is the times between from and to taken by events, joined where
// they touch, in order. An event with no end takes an hour; one that
// lasts all day takes none, as a birthday does not.
func (b *Book) Busy(from, to time.Time) []Span {
	if _, ok := b.Store.Types().Get("event"); !ok {
		return nil
	}
	events, _ := b.Store.List("event", store.ListOptions{})
	now := b.Now()
	var out []Span
	for _, e := range events {
		starts, _ := e.Fields["starts"].(string)
		at, day, ok := when.Stored(starts, now)
		if !ok || day {
			continue
		}
		long := time.Hour
		if ends, _ := e.Fields["ends"].(string); ends != "" {
			if end, _, ok := when.Stored(ends, now); ok && end.After(at) {
				long = end.Sub(at)
			}
		}
		times := []time.Time{at}
		if repeat, _ := e.Fields["repeat"].(string); repeat != "" {
			for _, v := range when.Occurrences(repeat, starts, from.Add(-long).Format("2006-01-02"), to.Format("2006-01-02"), busyMost, now) {
				if t, _, ok := when.Stored(v, now); ok {
					times = append(times, t)
				}
			}
		}
		for _, t := range times {
			s := Span{From: t, To: t.Add(long)}
			if !s.From.Before(to) || !s.To.After(from) {
				continue
			}
			if s.From.Before(from) {
				s.From = from
			}
			if s.To.After(to) {
				s.To = to
			}
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].From.Before(out[j].From) })
	var joined []Span
	for _, s := range out {
		if n := len(joined); n > 0 && !s.From.After(joined[n-1].To) {
			if s.To.After(joined[n-1].To) {
				joined[n-1].To = s.To
			}
			continue
		}
		joined = append(joined, s)
	}
	return joined
}
