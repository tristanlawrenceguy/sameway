package records

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An event at 9am on Thursday, moved to "October 25", landed at midnight:
// a day alone was stored as a whole day, and the time it had was lost
// without a word. A person moving something to another day means the same
// time on that day. So a day said in words (October 25, friday, next
// week), written over a moment, keeps the moment's time of day, from every
// way in. A date written as one, 2026-10-25, is a whole day as it says:
// it is what a page sends when the person clears the time.

// dayKeepsTime gives each day-only value written over a moment the time
// of day the moment had. now is the record's fields before the change.
func dayKeepsTime(t *schema.Type, fields, now map[string]any, clock time.Time) {
	for name, v := range fields {
		f, ok := t.Field(name)
		s, isText := v.(string)
		if !ok || f.Type != "datetime" || !isText || s == "" || writtenDate(s) {
			continue
		}
		old, _ := now[name].(string)
		was, wasDay, ok := when.Stored(old, clock)
		if !ok || wasDay {
			continue
		}
		day, isDay, ok := when.Parse(s, clock)
		if !ok || !isDay {
			continue
		}
		was = was.In(clock.Location())
		fields[name] = when.Store(time.Date(day.Year(), day.Month(), day.Day(), was.Hour(), was.Minute(), 0, 0, clock.Location()), false)
	}
}

// writtenDate says a value is a date written as one, 2026-10-25, as a page
// sends it, rather than a day said in words.
func writtenDate(s string) bool {
	if when.IsDay(s) {
		return true
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
