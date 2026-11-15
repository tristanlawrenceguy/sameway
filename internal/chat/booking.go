package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A request for a time, made by anyone through the booking link
// (server/book.go), is a suggested event on Today and nothing more: it
// waits for the person's Make it, Change or No, each on its own, and
// nothing is booked until they press one. What the asker wrote is kept
// as data in the event's notes and shown to the person; nothing acts on
// it.

// ByBooking marks a suggestion that came through the booking link.
const ByBooking = "booking"

// Booking puts a request for an hour from at to the person, as a
// suggested event.
func (s *Service) Booking(name, email, note string, at time.Time) error {
	name, email, note = trim.Line(name, 80), trim.Line(email, 120), trim.Clip(strings.TrimSpace(note), 500)
	if name == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("say your name and your email, so they can answer")
	}
	words := s.spanOf(at)
	notes := fmt.Sprintf("Asked for through your booking link by %s (%s).", name, email)
	if note != "" {
		notes += "\n\nThey wrote:\n\n> " + strings.ReplaceAll(note, "\n", "\n> ")
	}
	fields := map[string]any{"title": "Meeting with " + name, "starts": when.Store(at, false), "ends": when.Store(at.Add(time.Hour), false), "notes": notes}
	summary := fmt.Sprintf("%s (%s) asks for a time with you: %s. Book it?", name, email, words)
	_, err := s.Apply(records.Op{Type: records.ProposalType, After: map[string]any{"summary": summary, "detail": "Through your booking link",
		"action": map[string]any{"tool": "create_record", "type": "event", "fields": fields}, "state": "pending",
		"by_action": ByBooking, "yes": "Book it", "no": "No"}})
	return err
}

// spanOf is an hour from at, as a person reads it.
func (s *Service) spanOf(at time.Time) string {
	at = at.In(s.Now().Location())
	return at.Format("Mon 2 Jan") + ", " + when.Clock(at, s.H24()) + " to " + when.Clock(at.Add(time.Hour), s.H24())
}
