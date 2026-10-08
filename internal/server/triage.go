package server

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What comes in to sort (email, a message shared from a phone) is put to
// the model as it comes (chat/triage.go), in the background, one at a time:
// what it asks of the person becomes a suggestion on Today, kept with a
// press; what asks nothing, a newsletter or a receipt, is filed away as a
// note, logged, Undo bringing it back. With no model, or one that fails,
// nothing changes and the person sorts it by hand, as before.

var triageNow = make(chan struct{}, 1)

// triageSoon wakes the triage for something new.
func (s *Server) triageSoon() {
	select {
	case triageNow <- struct{}{}:
	default:
	}
}

// KeepTriage sorts what waits to be sorted, as it comes and every few
// minutes for what a model was not there for.
func (s *Server) KeepTriage(ctx context.Context) {
	go func() {
		tick := time.NewTicker(5 * time.Minute)
		defer tick.Stop()
		for {
			s.triageWaiting(ctx)
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			case <-triageNow:
			}
		}
	}()
}

// triageWaiting puts each note to sort that has no suggestion yet to the
// model, and says how many it sorted.
func (s *Server) triageWaiting(ctx context.Context) int {
	if s.app.Chat.Provider == nil {
		return 0
	}
	n := 0
	for _, note := range s.mailToSort() {
		if s.suggestionFor(note.ID) != nil {
			continue
		}
		c, cancel := context.WithTimeout(ctx, 3*time.Minute)
		err := s.triageNote(c, note)
		cancel()
		if err != nil {
			log.Printf("triage: %v", err)
			return n // the model is away or failing; next round
		}
		n++
	}
	return n
}

var sentLine = regexp.MustCompile(`^From .*, (\w{3} \d{1,2} \w{3} \d{4} \d{2}:\d{2})`)

// triageNote asks what one note asks of the person and keeps the answer;
// one that asks nothing is filed away.
func (s *Server) triageNote(ctx context.Context, note *store.Record) error {
	title, _ := note.Fields["title"].(string)
	body, _ := note.Fields["body"].(string)
	sent := note.CreatedAt.Local()
	if m := sentLine.FindStringSubmatch(body); m != nil {
		if at, err := time.ParseInLocation("Mon 2 Jan 2006 15:04", m[1], time.Local); err == nil {
			sent = at
		}
	}
	sug, err := s.app.Chat.Triage(ctx, title+"\n\n"+body, sent, s.peopleNames())
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(sug)
	s.app.Store.SetMeta("triage:"+note.ID, string(raw))
	if sug.Task {
		return nil
	}
	_, _, err = records.WriteAs(s.app.Store, records.Who{Actor: "assistant", Via: "sorting what came in"}, "updated", "note", note.ID, map[string]any{"tags": withoutTag(note, toSort)})
	return err
}

// suggestionFor is what the model made of a note, or nil before it has.
func (s *Server) suggestionFor(id string) *chat.Suggestion {
	raw := s.app.Store.Meta("triage:" + id)
	if raw == "" {
		return nil
	}
	var sug chat.Suggestion
	if json.Unmarshal([]byte(raw), &sug) != nil {
		return nil
	}
	return &sug
}

func (s *Server) peopleNames() []string {
	people, _ := s.app.Store.List(records.PersonType, store.ListOptions{})
	var out []string
	for _, p := range people {
		if n, _ := p.Fields["name"].(string); strings.TrimSpace(n) != "" {
			out = append(out, n)
		}
	}
	return out
}

func withoutTag(r *store.Record, tag string) []any {
	tags, _ := r.Fields["tags"].([]any)
	var out []any
	for _, t := range tags {
		if t != tag {
			out = append(out, t)
		}
	}
	return out
}
