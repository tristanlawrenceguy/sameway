package search

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// "Everything on Thursday" was searched as words, "Thursday 8 October
// 2026", and nothing has those words: what is on a day says so in its
// date, not its text. A search that is a day finds what falls on it: a
// task due, an event that starts, a reminder that rings. A weekday alone
// asked on that weekday is today and the coming one, either of which the
// person may mean.

// filler is said around a day without changing it: "on Thursday", "due
// tomorrow", "everything on Friday".
var dayFiller = map[string]bool{"on": true, "due": true, "for": true, "everything": true, "what": true, "is": true, "s": true, "whats": true, "all": true, "at": true}

// daysAsked are the days a query asks for, when it is a day and nothing
// else.
func daysAsked(q string, now time.Time) []time.Time {
	var kept []string
	for _, w := range strings.Fields(strings.ToLower(strings.NewReplacer(",", " ", "'", " ", "?", " ").Replace(q))) {
		if !dayFiller[w] {
			kept = append(kept, w)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	phrase := strings.Join(kept, " ")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if len(kept) == 1 {
		for name, wd := range weekdayNames {
			if phrase == name {
				coming := today.AddDate(0, 0, (int(wd)-int(now.Weekday())+7)%7)
				if coming.Equal(today) {
					return []time.Time{today, today.AddDate(0, 0, 7)}
				}
				return []time.Time{coming}
			}
		}
	}
	at, day, ok := when.Parse(phrase, now)
	if !ok || !day {
		return nil
	}
	return []time.Time{at}
}

var weekdayNames = map[string]time.Weekday{"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday}

// onDays is every record with a date on one of the days, the day said.
func onDays(st *store.Store, types *schema.Set, days []time.Time, now time.Time) []Hit {
	want := map[string]bool{}
	for _, d := range days {
		want[d.Format("2006-01-02")] = true
	}
	var hits []Hit
	for _, t := range types.Types {
		if Skip[t.Name] || t.Internal {
			continue
		}
		var dated []schema.Field
		for _, f := range t.Fields {
			if f.Type == "datetime" {
				dated = append(dated, f)
			}
		}
		if len(dated) == 0 {
			continue
		}
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			for _, f := range dated {
				v, _ := rec.Fields[f.Name].(string)
				at, allDay, ok := when.Parse(v, now)
				if v == "" || !ok {
					continue
				}
				at = at.In(time.Local)
				if !want[at.Format("2006-01-02")] {
					continue
				}
				said := schema.Words(f.Name) + " " + at.Format("Monday 2 January")
				if !allDay {
					said += at.Format(" 15:04")
				}
				title, _ := texts(t, rec)
				hits = append(hits, Hit{Type: t.Name, ID: rec.ID, Title: title, Snippet: said, Href: "/t/" + t.Name + "/" + rec.ID})
				break
			}
		}
	}
	return hits
}
