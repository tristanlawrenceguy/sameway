package export

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Reply writes the answer to an invitation someone sent (METHOD:REPLY,
// RFC 5546): the event by their id for it, from the one answering, yes or
// no, which their calendar takes as Gmail's or Outlook's own answer.
func Reply(w io.Writer, t *schema.Type, rec *store.Record, organizer, attendee string, yes bool) error {
	state := "DECLINED"
	if yes {
		state = "ACCEPTED"
	}
	ev := events(t, []*store.Record{rec}, time.Now().UTC().Format("20060102T150405Z")) // wall clock: when the file is made
	if len(ev) == 0 {
		return fmt.Errorf("the invitation has no day")
	}
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//sameway//sameway//EN", "CALSCALE:GREGORIAN", "METHOD:REPLY"}
	for _, l := range ev[:len(ev)-1] {
		if !strings.HasPrefix(l, "STATUS:") && !strings.HasPrefix(l, "X-SAMEWAY-DONE") && !strings.HasPrefix(l, "DESCRIPTION:") {
			lines = append(lines, l)
		}
	}
	lines = append(lines, "ORGANIZER"+person(organizer), "ATTENDEE;PARTSTAT="+state+person(attendee), "END:VEVENT", "END:VCALENDAR")
	return writeLines(w, lines)
}
