package chat

import (
	"fmt"
	"sort"
	"strings"
	"time"

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
func (s *Service) datesSaid(t *schema.Type, given map[string]any, rec *store.Record) string {
	now := s.clock()
	var said []string
	on := map[time.Weekday]bool{}
	for _, f := range t.Fields {
		if f.Type != "datetime" {
			continue
		}
		if _, ok := given[f.Name]; !ok {
			continue
		}
		v, _ := rec.Fields[f.Name].(string)
		at, day, ok := when.Parse(v, now)
		if v == "" || !ok {
			continue
		}
		at = at.In(time.Local) // as the store reads a time without a zone
		if day {
			said = append(said, f.Name+" "+at.Format("Monday 2 January 2006"))
		} else {
			said = append(said, f.Name+" "+at.Format("Monday 2 January 2006, 15:04"))
		}
		on[at.Weekday()] = true
	}
	if len(said) == 0 {
		return ""
	}
	text := " Its days: " + strings.Join(said, "; ") + "."
	named := s.weekdaysSaid()
	for _, wd := range named {
		if on[wd] {
			return text
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

// weekdaysSaid is the weekdays the person named in their latest message.
func (s *Service) weekdaysSaid() []time.Weekday {
	msgs, _ := s.Store.List(MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 10})
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
