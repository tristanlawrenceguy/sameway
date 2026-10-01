package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A functionality appears when there is a reason, and the reason is said
// (state/vision.md): either the person asked for it, or Sameway has
// watched enough to offer it and can say why in one sentence, as something
// that can be declined and taken back. This is the watching.
//
// A part of a page is off until asked for (parts.go). Asked for on the
// same kind of page often enough, the address three times in a week, it is
// offered for good: "You have opened every field on a task's page 3 times
// this week." Yes turns it on for the workspace (ui.show), in the log and
// undoable; No is remembered, and it is not asked again. Only the owner is
// watched, and only for the owner's own settings; nothing leaves the
// workspace.

const (
	offerAfter  = 3
	offerWithin = 7 * 24 * time.Hour
)

// noticeUse notes which parts this view was asked to show, and offers one
// for good once it has been asked for often enough. It returns the offer
// when one was made just now, for the page to put in front of them.
func (s *Server) noticeUse(r *http.Request, t *schema.Type, rec *store.Record, always, here []string) *store.Record {
	if !chat.VisitorOf(r.Context()).Owner() || pageAction(r) || r.Header.Get("X-Requested-With") == "sameway-live" {
		return nil
	}
	var offered *store.Record
	for _, key := range here {
		if key == "" || has(always, key) {
			continue
		}
		asked := s.askedFor(key, t.Name)
		if len(asked) < offerAfter || s.app.Store.Meta("offered:show:"+key) != "" {
			continue
		}
		part := partWords(s, t, rec, key)
		ask := fmt.Sprintf("Show %s on every page it fits?", part)
		why := fmt.Sprintf("You have opened %s on %s %d times this week.", part, a(schema.Words(t.Name))+"'s page", len(asked))
		p, err := s.app.Chat.Offer(ask, why, "Show it every time", "Not now", map[string]any{"tool": "set_setting", "key": "ui.show", "value": "+" + key})
		if err != nil {
			continue
		}
		// Asked once: a No is an answer, and a Yes is in the log.
		s.app.Store.SetMeta("offered:show:"+key, time.Now().UTC().Format(time.RFC3339))
		offered = p
	}
	return offered
}

// askedFor notes one more asking for a part on a kind of page and says
// when it was asked for within the week, this time included.
func (s *Server) askedFor(key, typ string) []time.Time {
	metaKey := "use:show:" + key + ":" + typ
	var times []time.Time
	json.Unmarshal([]byte(s.app.Store.Meta(metaKey)), &times)
	now := time.Now().UTC()
	kept := []time.Time{now}
	for _, at := range times {
		if now.Sub(at) < offerWithin {
			kept = append(kept, at)
		}
	}
	raw, _ := json.Marshal(kept)
	s.app.Store.SetMeta(metaKey, string(raw))
	return kept
}

// partWords is a part as the question says it.
func partWords(s *Server, t *schema.Type, rec *store.Record, key string) string {
	switch key {
	case FieldsPart:
		return "every field"
	case RemindPart:
		return "the way to set a reminder"
	case AskPart:
		return "the way to ask the assistant about it"
	case DayPart:
		return "its day on the calendar"
	case ContentsPart:
		return "its contents"
	case PlacePart:
		return "where it is in its piece"
	case OutlinePart:
		return "its outline"
	case MaterialPart:
		return "the material kept with it"
	case RecordingPart:
		return "its recording"
	case WriteUpPart:
		return "the offer to write it up"
	}
	for _, l := range relate.Of(s.app.Store, t, rec, time.Now()) {
		if l.Key == key {
			return strings.ToLower(schema.Plural(l.Type)) + " " + l.Why
		}
	}
	return "that part"
}

// a is a noun with its article: a task, an event.
func a(noun string) string {
	if noun != "" && strings.ContainsRune("aeiou", rune(strings.ToLower(noun)[0])) {
		return "an " + noun
	}
	return "a " + noun
}
