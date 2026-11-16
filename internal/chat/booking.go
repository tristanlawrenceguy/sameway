package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A time asked for through the booking link (server/book.go) is a request
// like any other (requests_made.go): an event on Today that waits for the
// person's Make it, Change or No, each on its own, with an address that
// tells the one asking what became of it. What the asker wrote is kept
// as data in the event's notes and shown to the person; nothing acts on
// it.

// Booking puts a request for an hour from at to the person, as an event,
// and gives the request's key.
func (s *Service) Booking(name, email, note string, at time.Time) (string, error) {
	name, email, note = trim.Line(name, 80), trim.Line(email, 120), trim.Clip(strings.TrimSpace(note), 500)
	if name == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("say your name and your email, so they know who asks")
	}
	notes := fmt.Sprintf("Asked for through your booking link by %s (%s).", name, email)
	if note != "" {
		notes += "\n\nThey wrote:\n\n> " + strings.ReplaceAll(note, "\n", "\n> ")
	}
	title := "Meeting with " + name
	fields := map[string]any{"title": title, "starts": when.Store(at, false), "ends": when.Store(at.Add(time.Hour), false), "notes": notes}
	what := fmt.Sprintf("make an event: %q, %s", title, s.spanOf(at))
	if note != "" {
		what += fmt.Sprintf(", about %q", trim.Line(note, 120))
	}
	// Who they say they are is theirs to say: a name, never a person here.
	asker := s.For(records.Visitor{Access: records.Public, Name: name + " (" + email + ")"})
	summary, token, err := asker.putRequest(map[string]any{"tool": "create_record", "type": "event", "fields": fields}, what, "Through your booking link")
	if err == nil && s.Requested != nil {
		s.Requested(summary)
	}
	return token, err
}

// spanOf is an hour from at, as a person reads it.
func (s *Service) spanOf(at time.Time) string {
	at = at.In(s.Now().Location())
	return at.Format("Mon 2 Jan") + ", " + when.Clock(at, s.H24()) + " to " + when.Clock(at.Add(time.Hour), s.H24())
}
