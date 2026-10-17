package records

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An agent moving a clean-up to "9am on the 25th" wrote 09:00:00Z: UTC,
// which where the person lives is 10am, and the card it then wrote for
// twelve volunteers said 9am. Every moment is kept in UTC, so it is told
// as it is written, before it is kept: a time written with Z (or +00:00)
// that the person reads as another clock time is said back in their time,
// with how to write the time meant. Only a time as a person sets one, on
// the hour or a quarter of it with no seconds: a machine's now, 09:50:13Z,
// is the moment it says. It comes with the change's notes (notes.go), for
// every way in.

// utcNoticed says each time in fields written in UTC that the person, on
// clock, reads as another hour.
func utcNoticed(t *schema.Type, fields map[string]any, clock time.Time) []string {
	var out []string
	for name, v := range fields {
		f, ok := t.Field(name)
		s, isText := v.(string)
		if !ok || f.Type != "datetime" || !isText || !(strings.HasSuffix(s, "Z") || strings.HasSuffix(s, "+00:00")) || when.IsDay(s) {
			continue
		}
		at, err := time.Parse(time.RFC3339, s)
		if err != nil {
			continue
		}
		local := at.In(clock.Location())
		if local.Format("15:04") == at.UTC().Format("15:04") || !typed(at) {
			continue
		}
		out = append(out, fmt.Sprintf("%s was written in UTC (%s), which for the person is %s. To mean %s their time, write %s.",
			t.FieldWords(name), at.UTC().Format("15:04"), when.At(local, clock, false), at.UTC().Format("15:04"), at.UTC().Format("2006-01-02T15:04")))
	}
	return out
}

// typed says a time is one a person would set: whole minutes on a quarter
// hour, no seconds.
func typed(at time.Time) bool {
	return at.Second() == 0 && at.Nanosecond() == 0 && at.Minute()%15 == 0
}
