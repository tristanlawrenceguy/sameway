package chat

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A small model read "2026-10-15T00:00:00Z" as a Saturday, and moved a
// 10:00 dentist to "10:00Z", two hours out here: it counts weekdays badly
// and does not turn UTC into this computer's time. So what it reads of a
// record says each day in words, with its weekday and the time here, as
// the person would say it, and how to write one back.

// daysOf is a record's dates in words: "due Thursday 15 October", "starts
// Thursday 15 October 12:30". Empty when it has none.
func daysOf(t *schema.Type, rec *store.Record, now time.Time) []string {
	var out []string
	for _, f := range t.Fields {
		if f.Type != "datetime" {
			continue
		}
		v, _ := rec.Fields[f.Name].(string)
		at, allDay, ok := when.Parse(v, now)
		if v == "" || !ok {
			continue
		}
		at = at.In(time.Local)
		day := at.Format("Monday 2 January")
		if at.Year() != now.Year() {
			day += at.Format(" 2006")
		}
		if !allDay {
			day += at.Format(" 15:04")
		}
		out = append(out, schema.Words(f.Name)+" "+day)
	}
	return out
}

// daysLine is daysOf as get_record says it, with how to write a time back
// so it stays the time here.
func daysLine(t *schema.Type, rec *store.Record, now time.Time) string {
	days := daysOf(t, rec, now)
	if len(days) == 0 {
		return ""
	}
	return strings.Join(days, "; ") + " (this computer's time; to keep a time as it is here, write it without Z, as 2026-10-20T10:00: a time ending in Z is UTC)"
}
