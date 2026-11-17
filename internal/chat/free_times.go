package chat

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// When the owner is free, as anyone arranging to meet them by hand would
// say it: the open stretches of each working day, worked out from the
// events, never what fills the rest. The owner asks it to offer times;
// someone writing to the owner in a conversation by email asks it to pick
// one (email_conversation.go). Nobody else from outside reads it.

const (
	freeDays  = 14 // how far ahead, unless asked
	freeFirst = 9  // the working day, in the owner's own zone
	freeLast  = 17
)

var freeTimesOp = Op{Title: "Say when the owner is free",
	Access: ForViewers, Traits: Traits{ReadOnly: true, Idempotent: true},
	Words: []string{"free", "available", "availability", "when can", "time to meet", "propose", "suggest a time"},
	Doing: saying("Looking at the calendar"),
	Run:   func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.freeTimes(call.Args) },
	Tool: llm.Tool{Name: "free_times",
		Description: "When the owner is free to meet, working hours of each day, from their calendar: the open stretches only, never what is in the rest. To offer times, or to pick one someone asked for. from and to are days (2026-11-23); the next two weeks unless said.",
		Schema: obj(map[string]any{
			"from": map[string]any{"type": "string"},
			"to":   map[string]any{"type": "string"},
		})}}

// mayKnowFree says whether whoever this service speaks for may know when
// the owner is free: anyone let in, and someone in a conversation by
// email the owner began; not a stranger on the published pages.
func (s *Service) mayKnowFree() bool {
	return s.who.Access != records.Public || s.who.Device == ByEmail
}

func (s *Service) freeTimes(raw json.RawMessage) toolResult {
	if !s.mayKnowFree() {
		return fail("when the owner is free is theirs to say; ask them for a time with create_record instead")
	}
	var a struct{ From, To string }
	json.Unmarshal(raw, &a)
	loc := s.Now().Location()
	from := s.Now().Truncate(time.Hour).Add(time.Hour)
	if d, err := time.ParseInLocation("2006-01-02", a.From, loc); err == nil && d.After(from) {
		from = d
	}
	to := from.AddDate(0, 0, freeDays)
	if d, err := time.ParseInLocation("2006-01-02", a.To, loc); err == nil && d.After(from) && d.Before(from.AddDate(0, 2, 0)) {
		to = d.AddDate(0, 0, 1)
	}
	busy := s.Busy(from, to)
	var days []string
	for day := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc); day.Before(to); day = day.AddDate(0, 0, 1) {
		start, end := day.Add(freeFirst*time.Hour), day.Add(freeLast*time.Hour)
		if start.Before(from) {
			start = from
		}
		if open := freeIn(start, end, busy); len(open) > 0 {
			days = append(days, day.Format("Mon 2 Jan")+": "+s.stretches(open))
		}
	}
	if len(days) == 0 {
		return toolResult{text: "No free time in working hours then."}
	}
	return toolResult{text: "Free, in working hours (" + loc.String() + "):\n" + strings.Join(days, "\n")}
}

// freeIn are the stretches of start to end no busy span covers.
func freeIn(start, end time.Time, busy []records.Span) []records.Span {
	var out []records.Span
	at := start
	for at.Before(end) {
		next := end
		blocked := false
		for _, b := range busy {
			if !at.Before(b.From) && at.Before(b.To) {
				at, blocked = b.To, true
				break
			}
			if b.From.After(at) && b.From.Before(next) {
				next = b.From
			}
		}
		if blocked {
			continue
		}
		out = append(out, records.Span{From: at, To: next})
		at = next
	}
	return out
}

// stretches are free spans as a person says them: 9am to 12pm, 2pm to 5pm.
func (s *Service) stretches(open []records.Span) string {
	var parts []string
	for _, o := range open {
		parts = append(parts, when.Clock(o.From, s.H24())+" to "+when.Clock(o.To, s.H24()))
	}
	return strings.Join(parts, ", ")
}
