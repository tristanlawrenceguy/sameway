package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A meeting the person records asks to be recorded when it starts: a
// reminder at its start, repeating as it does, which rings like any other
// (on the computer, on open pages, with Snooze) and opens the meeting's
// page with its recording part, ready to record. Nothing new rings: it is
// the clock's own reminder, made for this. Only a meeting someone wants
// recorded asks; a calendar full of events stays quiet.

// ReminderType is the content type of an alarm.
const ReminderType = "reminder"

// RecordAbout is the page a meeting's reminder leads to.
func RecordAbout(eventID string) string {
	return "/t/" + EventType + "/" + eventID + "?show=recording"
}

func (s *Service) recordingTools() []llm.Tool {
	if _, ok := s.Store.Types().Get(ReminderType); !ok {
		return nil
	}
	if _, ok := s.Store.Types().Get(EventType); !ok {
		return nil
	}
	return []llm.Tool{{Name: "ask_to_record",
		Description: "Have a meeting ask to be recorded when it starts: a reminder at its start, repeating as it does, that opens its page ready to record (the microphone, and the computer's sound for a call). Use it when the person wants a meeting recorded, or records this one each time; never for every event.",
		Schema: obj(map[string]any{
			"event": map[string]any{"type": "string", "description": "The event id of the meeting."},
		}, "event")}}
}

func (s *Service) askToRecord(eventID string, now time.Time) toolResult {
	ev, err := s.Store.Get(EventType, eventID)
	if err != nil {
		return fail("no event %s; find_records on event", eventID)
	}
	et, _ := s.Store.Types().Get(EventType)
	title := Name(s.Store, et, ev)
	about := RecordAbout(eventID)
	if had := remindersAbout(s.Store, about); len(had) > 0 {
		return toolResult{text: fmt.Sprintf("%s already asks to be recorded when it starts.", title)}
	}
	starts, _ := ev.Fields["starts"].(string)
	repeat, _ := ev.Fields["repeat"].(string)
	at := starts
	if at == "" {
		return fail("%s has no start time to ring at; set starts first", title)
	}
	// A meeting that repeats rings at its next time still to come.
	for i := 0; repeat != "" && i < 1000; i++ {
		ts, err := time.Parse(time.RFC3339, at)
		if err != nil || !ts.Before(now) {
			break
		}
		next, ok := when.Next(repeat, at, now)
		if !ok {
			break
		}
		at = next
	}
	if ts, err := time.Parse(time.RFC3339, at); err == nil && ts.Before(now) {
		return fail("%s started at %s and does not repeat, so there is nothing to ring for", title, at)
	}
	rec, c, err := Write(s.Store, "created", ReminderType, "", map[string]any{
		"title": "Record " + title, "at": at, "repeat": repeat, "about": about,
		"notes": "It opens the meeting's page ready to record."})
	if err != nil {
		return fail("could not set it: %v", err)
	}
	return toolResult{text: fmt.Sprintf("%s now asks to be recorded when it starts (%s%s), with a reminder that opens its page ready to record: /t/%s/%s.",
		title, at, map[bool]string{true: ", " + strings.ToLower(when.RepeatText(repeat)), false: ""}[repeat != ""], ReminderType, rec.ID), change: &c}
}

func remindersAbout(st *store.Store, about string) []*store.Record {
	all, _ := st.List(ReminderType, store.ListOptions{})
	var out []*store.Record
	for _, r := range all {
		if r.Fields["about"] == about && r.Fields["state"] != "done" {
			out = append(out, r)
		}
	}
	return out
}
