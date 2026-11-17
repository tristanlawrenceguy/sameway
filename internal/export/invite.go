package export

import (
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An invitation is an event sent as a calendar that asks (METHOD:REQUEST,
// RFC 5546): Gmail, Outlook and Apple Mail show it with their own Yes, No
// and Maybe, put it in the calendar of whoever gets it, and send their
// answer back to the organiser. Sent again with a higher sequence it
// moves the event there; sent as CANCEL it takes it away.

// Invite writes one event as an invitation from organizer to attendees,
// each "Name <email>" or an address, or with cancel as its cancelling.
func Invite(w io.Writer, t *schema.Type, rec *store.Record, organizer string, attendees []string, sequence int, cancel bool) error {
	method, status := "REQUEST", "CONFIRMED"
	if cancel {
		method, status = "CANCEL", "CANCELLED"
	}
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//sameway//sameway//EN", "CALSCALE:GREGORIAN", "METHOD:" + method}
	ev := events(t, []*store.Record{rec}, time.Now().UTC().Format("20060102T150405Z")) // wall clock: when the file is made
	if len(ev) == 0 {
		return fmt.Errorf("%s has no day to send", t.Called(rec.ID, rec.Fields))
	}
	var who []string
	who = append(who, "ORGANIZER"+person(organizer))
	for _, a := range attendees {
		who = append(who, "ATTENDEE;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE"+person(a))
	}
	who = append(who, fmt.Sprintf("SEQUENCE:%d", sequence), "STATUS:"+status)
	for _, l := range ev[:len(ev)-1] {
		if !strings.HasPrefix(l, "STATUS:") && !strings.HasPrefix(l, "X-SAMEWAY-DONE") {
			lines = append(lines, l)
		}
	}
	lines = append(append(lines, who...), "END:VEVENT", "END:VCALENDAR")
	return writeLines(w, lines)
}

// person is someone as ORGANIZER and ATTENDEE take them: ;CN=name:mailto:address.
func person(who string) string {
	a, err := mail.ParseAddress(who)
	if err != nil {
		return ":mailto:" + strings.TrimSpace(who)
	}
	if a.Name == "" {
		return ":mailto:" + a.Address
	}
	return ";CN=" + strings.NewReplacer(`"`, "", ";", "", ":", "", ",", "").Replace(a.Name) + ":mailto:" + a.Address
}
