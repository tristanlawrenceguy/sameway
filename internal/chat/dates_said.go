package chat

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A model counts days badly: asked for next Tuesday on a Friday, it wrote
// the 7th, a Wednesday, two times in three. Sameway counts instead. Each
// day a record is given is said back with its weekday, and when the
// person named a weekday and nothing given falls on it, the result says
// so and names the days that do, for the model to put right.

var weekdayNames = map[string]time.Weekday{"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday}

// datesSaid is what a write did to the record's days, in words, or "".
// moved says the record had days before, which a weekday the person named
// is counted from ("move Thursday's to Friday"), not from today.
func (s *Service) datesSaid(t *schema.Type, given map[string]any, rec *store.Record, moved bool) string {
	now := s.clock()
	var said []string
	var ats []time.Time
	on := map[time.Weekday]bool{}
	for _, f := range t.Fields {
		if f.Type != "datetime" {
			continue
		}
		if _, ok := given[f.Name]; !ok {
			continue
		}
		v, _ := rec.Fields[f.Name].(string)
		at, day, ok := when.Stored(v, now.In(time.Local)) // a whole day as its date, a moment here
		if v == "" || !ok {
			continue
		}
		if day {
			said = append(said, f.Name+" "+at.Format("Monday 2 January 2006"))
		} else {
			said = append(said, f.Name+" "+at.Format("Monday 2 January 2006, 15:04"))
		}
		on[at.Weekday()] = true
		ats = append(ats, at)
	}
	if len(said) == 0 {
		return ""
	}
	text := " Its days: " + strings.Join(said, "; ") + "."
	named := s.weekdaysSaid()
	for _, wd := range named {
		if on[wd] {
			if moved {
				return text
			}
			return text + s.notTheComing(ats, named, now)
		}
	}
	if len(named) > 0 {
		var names, days []string
		for _, wd := range named {
			first := now.AddDate(0, 0, (int(wd)-int(now.Weekday())+7)%7)
			names = append(names, wd.String())
			days = append(days, fmt.Sprintf("the coming %s is %s, the one after %s", wd, first.Format("2 January"), first.AddDate(0, 0, 7).Format("2 January")))
		}
		text += fmt.Sprintf(" The person said %s, and none of these is one: %s. If they meant one of those, put it right with update_record.", strings.Join(names, " or "), strings.Join(days, "; "))
	}
	return text
}

// notTheComing says when a day falls on a weekday the person named but is
// not the coming one: asked on a Tuesday to plan "a walk on Saturday", a
// small model wrote the Saturday after next. It may be what they meant
// (next Saturday), so it is said, not changed.
func (s *Service) notTheComing(ats []time.Time, named []time.Weekday, now time.Time) string {
	var out []string
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, at := range ats {
		for _, wd := range named {
			if at.Weekday() != wd {
				continue
			}
			if ahead := int(at.Sub(today).Hours() / 24); ahead >= 7 {
				first := today.AddDate(0, 0, (int(wd)-int(now.Weekday())+7)%7)
				out = append(out, fmt.Sprintf("%s is not the coming %s, which is %s", at.Format("Monday 2 January"), wd, first.Format("2 January")))
			}
		}
	}
	if len(out) == 0 {
		return ""
	}
	return " " + strings.Join(out, "; ") + ". If the person meant the coming one, put it right with update_record."
}

// weekdaysSaid is the weekdays the person named in their latest message.
func (s *Service) weekdaysSaid() []time.Weekday {
	msgs, _ := s.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 10})
	for _, m := range msgs {
		if m.Fields["role"] != "user" {
			continue
		}
		words := strings.FieldsFunc(strings.ToLower(fmt.Sprint(m.Fields["content"])), func(r rune) bool { return r < 'a' || r > 'z' })
		seen := map[time.Weekday]bool{}
		var out []time.Weekday
		for _, w := range words {
			if wd, ok := weekdayNames[w]; ok && !seen[wd] {
				seen[wd] = true
				out = append(out, wd)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
		return out
	}
	return nil
}

func (s *Service) clock() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// timeLost says when a change gave a timed field a day alone, so its time
// went: a model moving lunch at 12:30 to Friday wrote the day only, two
// times in three, and the lunch became all day.
func timeLost(t *schema.Type, given map[string]any, was, now *store.Record) string {
	var lost []string
	for _, f := range t.Fields {
		if _, ok := given[f.Name]; !ok || f.Type != "datetime" {
			continue
		}
		before, _ := was.Fields[f.Name].(string)
		after, _ := now.Fields[f.Name].(string)
		old, err := time.Parse(time.RFC3339, before)
		if err != nil || strings.HasSuffix(before, "T00:00:00Z") || !strings.HasSuffix(after, "T00:00:00Z") {
			continue
		}
		lost = append(lost, fmt.Sprintf("%s was at %s and is now the whole day", f.Name, old.In(time.Local).Format("15:04")))
	}
	if len(lost) == 0 {
		return ""
	}
	return " " + strings.Join(lost, "; ") + ": if it should keep its time, give the day with the time."
}
