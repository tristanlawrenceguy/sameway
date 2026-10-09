package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The clock: the time, and reminders. A reminder is a record (an alarm
// at a time, or a timer that ends at one) that the clock sets, lists,
// and rings when its time comes. Ringing happens here: every few seconds
// the server marks what is due as rung (ring.go), and a page with a clock
// hears of it on its one connection, /events?ring=1 (sync.go), and shows
// it, sounds, and notifies. A rung reminder stays until dismissed or given
// five more minutes, and it shows on the calendar like anything with a day.

// tellRings tells an open page, as server-sent events, each reminder that
// has rung and it has not been told of yet.
func (s *Server) tellRings(w io.Writer, told map[string]bool) {
	for _, rec := range s.ringing() {
		if told[rec.ID] {
			continue
		}
		told[rec.ID] = true
		t, _ := s.app.Types.Get(ReminderType)
		// What it is about, in words and as a link, as the machine's own
		// notification says it: Water, 3 of 8 glasses so far.
		text, link := blocks.RingWords(s.app.Store, rec)
		body, _ := json.Marshal(map[string]any{"id": rec.ID, "title": s.title(t, rec), "href": "/t/" + ReminderType + "/" + rec.ID, "text": text, "url": link})
		fmt.Fprintf(w, "event: ring\ndata: %s\n\n", body)
	}
}

// ringing is every reminder that has rung and not been dismissed.
func (s *Server) ringing() []*store.Record {
	t, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return nil
	}
	recs, err := query.Filter(s.app.Store, t, []string{"state=rang"}, "at", 0, s.now())
	if err != nil {
		return nil
	}
	return recs
}

// ring marks every reminder whose time has come as rung, once, and says
// which. The log has it too, as the system's doing.
func (s *Server) ring(now time.Time) []*store.Record {
	t, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return nil
	}
	recs, err := query.Filter(s.app.Store, t, []string{"state=set"}, "at", 0, now)
	if err != nil {
		return nil
	}
	rang := s.nudges(now)
	for _, rec := range recs {
		at, _ := rec.Fields["at"].(string)
		ts, err := time.Parse(time.RFC3339, at)
		if err != nil || ts.After(now) {
			continue
		}
		// Ringing is the system's: written as an op, logged as rang, and
		// not taken back.
		if _, err := records.ApplyOps(s.app.Store, records.Op{Type: ReminderType, ID: rec.ID, After: map[string]any{"state": "rang"}}); err != nil {
			log.Printf("clock: %v", err)
			continue
		}
		records.Record(s.app.Store, "system", records.Change{Action: "rang", Component: ReminderType, ID: rec.ID, Detail: s.title(t, rec)})
		rang = append(rang, rec)
	}
	return rang
}
