package blocks

import (
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The clock block: the time, what is ringing and what is coming. Setting,
// ringing and the stream an open page listens on are the server's
// (server/clock.go, ring.go).

// ClockComponent is the clock block.
const ClockComponent = "clock"

// resolveClock fills what the block leaves to the moment: the time, what
// is ringing, and what is coming.
func resolveClock(w *Workspace, props map[string]any, _ Place) map[string]any {
	out := copyProps(props)
	now := w.now()
	out["now"] = now.Format(time.RFC3339)
	out["time"] = when.Face(now, w.H24())
	out["date"] = now.Format("Monday 2 January")
	ringing := []any{}
	var next []coming
	if t, ok := w.Store.Types().Get(records.ReminderType); ok {
		recs, _ := query.Filter(w.Store, t, nil, "at", 0, now)
		for _, rec := range recs {
			item := map[string]any{"id": rec.ID, "title": records.Name(w.Store, t, rec), "href": "/t/" + records.ReminderType + "/" + rec.ID}
			if said := RepeatsOf(rec); said != "" {
				item["repeats"] = said
			}
			switch rec.Fields["state"] {
			case "rang":
				if text, _ := RingWords(w.Store, rec); rec.Fields["about"] != nil && text != item["title"] {
					item["text"] = text
				}
				ringing = append(ringing, item)
			case "set":
				v, _ := rec.Fields["at"].(string)
				at, err := time.Parse(time.RFC3339, v)
				if err != nil {
					continue
				}
				item["day"], item["time"] = DayOf(at, now), when.Clock(at.In(now.Location()), w.H24())
				item["kind"], _ = rec.Fields["kind"].(string)
				next = append(next, coming{at, item})
			}
		}
	}
	next = append(next, w.onToday(now)...)
	out["ringing"], out["upcoming"] = ringing, soonest(next, 8)
	return out
}

// coming is one thing in Coming up, with the moment it is sorted by.
type coming struct {
	at   time.Time
	item map[string]any
}

// soonest is what is coming in time order, a reminder and a task at the
// same hour side by side whatever they are, the first n of them.
func soonest(all []coming, n int) []any {
	sort.SliceStable(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })
	out := []any{}
	for i, c := range all {
		if i == n {
			break
		}
		out = append(out, c.item)
	}
	return out
}

// onToday is what falls today across every listed type with a day: the
// calendar's view of the day. A thing with no time of its own is all day,
// and comes first.
func (w *Workspace) onToday(now time.Time) []coming {
	day := now.Format("2006-01-02")
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var items []coming
	for _, t := range w.Store.Types().Types {
		if t.Internal || t.Name == records.ReminderType || !w.Listed(t) {
			continue
		}
		field := t.DayField()
		if field == "" {
			continue
		}
		recs, err := query.Filter(w.Store, t, nil, field, 0, now)
		if err != nil {
			continue
		}
		for _, rec := range recs {
			v, _ := rec.Fields[field].(string)
			ts, err := time.Parse(time.RFC3339, v)
			if err != nil {
				continue
			}
			item := map[string]any{"title": records.Name(w.Store, t, rec), "href": "/t/" + t.Name + "/" + rec.ID, "kind": "event", "time": "All day"}
			at := start
			if strings.HasSuffix(v, "T00:00:00Z") {
				if ts.UTC().Format("2006-01-02") != day {
					continue
				}
			} else {
				if ts.Local().Format("2006-01-02") != day {
					continue
				}
				at = ts.Local()
				item["time"] = when.Clock(at, w.H24())
			}
			items = append(items, coming{at, item})
		}
	}
	return items
}

// DayOf is the day of a moment in the words the clock uses beside its
// time: nothing for today, Tomorrow, a weekday within the week, a date
// after that.
func DayOf(at, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	at = at.In(now.Location())
	d := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, now.Location())
	switch diff := int(d.Sub(today).Hours() / 24); {
	case diff == 0:
		return ""
	case diff == 1:
		return "Tomorrow"
	case diff > 1 && diff < 7:
		return d.Format("Monday")
	}
	return d.Format("2 Jan")
}

// RepeatsOf is how often a reminder rings again, as the clock says it
// under its name: Repeats every Tuesday. Nothing for one that rings once.
func RepeatsOf(rec *store.Record) string {
	if rule, _ := rec.Fields["repeat"].(string); rule != "" {
		return "Repeats " + when.RepeatText(rule)
	}
	return ""
}
