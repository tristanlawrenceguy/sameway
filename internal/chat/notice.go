package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What the assistant's side knows to say about a change, said for every
// way in (records/notes.go): a record made that is called as open ones are
// (same_title.go), a person named and not set (whose_said.go), an action
// told when (action_when_said.go), an entry's habit (habit_said.go), and a
// block added over records another already shows (tools_similar.go). The assistant's tools, MCP and /api/tools say it
// through run; the JSON API through its writes.

// SayNotes registers what chat says of a change on its store, once for
// the store: a conversation of its own is another Service on it.
func (s *Service) SayNotes() {
	records.Notice(s.Store, noticeKey{}, s.notes)
}

// notes is what chat says of c.
func (s *Service) notes(c records.Change) []string {
	made := c.Made()
	out := s.copyNotes(c)                 // a block made or changed to hold records' words
	out = append(out, s.leftBehind(c)...) // left_behind.go: what stayed on a moved record's old day

	for _, op := range made {
		rec, err := s.Store.Get(op.Type, op.ID)
		if err != nil {
			continue
		}
		if op.Type == records.BlockType {
			name, _ := rec.Fields["component"].(string)
			props, _ := rec.Fields["props"].(map[string]any)
			canvas, _ := rec.Fields["canvas"].(string)
			if like := strings.TrimSpace(s.similarOn(canvas, name, props, rec.ID)); like != "" {
				out = append(out, like)
			}
			if crowded := s.crowdedSaid(rec); crowded != "" { // notice_crowded.go
				out = append(out, crowded)
			}
			continue
		}
		t, ok := s.Store.Types().Get(op.Type)
		if !ok || op.Type == records.MessageType || op.Type == records.ActivityType { // each note below checks the kinds it is for
			continue
		}
		// Of the record alone, so the same for every way in: one called as
		// open ones are, a person named and not made whose it is, an action
		// told when to run, an entry's habit as it stands now.
		said := strings.TrimSpace(s.sameTitle(t, rec) + s.whoseSaid(t, rec) + s.actionSaid(t, rec) + s.habitSaid(t, rec))
		if said == "" {
			continue
		}
		if len(made) > 1 { // which of several it is about
			said = fmt.Sprintf("%s %q: %s", rec.ID, t.Called(rec.ID, rec.Fields), said)
		}
		out = append(out, said)
	}
	return out
}

// copyNotes is what each block a change made or changed is told when it
// copies records as words: said with the change, first, not only as one
// line of Layout now's list, where an agent read past it.
func (s *Service) copyNotes(c records.Change) []string {
	var out []string
	for _, op := range c.Ops {
		if op.Type != records.BlockType || op.After == nil || op.ID == "" {
			continue
		}
		b, err := s.Store.Get(records.BlockType, op.ID)
		if err != nil {
			continue
		}
		for _, said := range copiesSaid(b, copiedBy(b, s.titleIndex([]*store.Record{b})), false) {
			out = append(out, fmt.Sprintf("This %s %s.", placedOf(b).called(), said))
		}
	}
	return out
}

// notesSaid is notes as they end a tool's answer, "" when there are none.
func notesSaid(notes []string) string {
	if len(notes) == 0 {
		return ""
	}
	return "\n" + strings.Join(notes, "\n")
}

// noticeKey is chat's one noticer on a store.
type noticeKey struct{}
