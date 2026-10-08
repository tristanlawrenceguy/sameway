package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A meeting the person records asks to be recorded when it starts: a
// reminder at its start, repeating as it does, which rings like any other
// (on the computer, on open pages, with Snooze) and opens the meeting's
// page with its recording part, ready to record. Nothing new rings: it is
// the clock's own reminder, made for this. Only a meeting someone wants
// recorded asks; a calendar full of events stays quiet.

// RecordAbout is the page a meeting's reminder leads to.
func RecordAbout(eventID string) string {
	return "/t/" + records.EventType + "/" + eventID + "?show=recording"
}

var recordingOps = []Op{{Title: "Record a meeting", Traits: Traits{Idempotent: true},
	Words: []string{"meeting", "record", "call", "zoom", "teams"},
	Doing: saying("Setting the meeting to ask to be recorded"),
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		return s.askToRecord(a.Event, a.How, s.clock())
	},
	Tool: llm.Tool{Name: "record_meeting",
		Description: "Record a meeting, transcribe it and write it up after: this sets it up. With how here, a reminder as it starts that opens its page ready to record (the microphone, and this computer's sound for a call); with how app, for a meeting Teams, Zoom or Meet records, a reminder as it ends to add that recording or transcript on its page. Each repeats as the meeting does. Use it when the person wants a meeting recorded, or records this one each time; never for every event.",
		Schema: obj(map[string]any{
			"event": map[string]any{"type": "string", "description": "The event id of the meeting."},
			"how":   map[string]any{"type": "string", "enum": []string{"here", "app"}, "description": "here: Sameway records it; app: the meeting app does, and its file is added after. here when left out."},
		}, "event")}, Offered: has(records.ReminderType, records.EventType)}}

func (s *Service) askToRecord(eventID, how string, now time.Time) toolResult {
	ev, err := s.Store.Get(records.EventType, eventID)
	if err != nil {
		return fail("no event %s; find_records on event", eventID)
	}
	et, _ := s.Store.Types().Get(records.EventType)
	title := records.Name(s.Store, et, ev)
	about := RecordAbout(eventID)
	name, notes := "Record "+title, "Starting now. Its page is ready to record: the microphone, and this computer's sound for a call."
	if how == "app" {
		name, notes = "Add the recording of "+title, "It has ended. Add the recording or transcript the meeting app made (a .vtt from Teams or Zoom, or the audio) on its page."
	}
	for _, r := range RemindersAbout(s.Store, about) {
		if r.Fields["title"] == name {
			return toolResult{text: fmt.Sprintf("%s already has that reminder: /t/%s/%s.", title, records.ReminderType, r.ID)}
		}
	}
	starts, _ := ev.Fields["starts"].(string)
	repeat, _ := ev.Fields["repeat"].(string)
	if starts == "" {
		return fail("%s has no start time to ring at; set starts first", title)
	}
	at := nextStart(starts, repeat, now, MeetingLength(ev))
	if how == "app" {
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			at = t.Add(MeetingLength(ev)).Format(time.RFC3339)
		}
	}
	if t, err := time.Parse(time.RFC3339, at); err == nil && t.Before(now) {
		return fail("%s is over and does not repeat, so there is nothing to ring for", title)
	}
	rec, c, err := records.Write(s.Store, "created", records.ReminderType, "", map[string]any{
		"title": name, "at": at, "repeat": repeat, "about": about, "notes": notes})
	if err != nil {
		return fail("could not set it: %v", err)
	}
	when := at
	if repeat != "" {
		when += ", " + strings.ToLower(whenRepeat(repeat))
	}
	return toolResult{text: fmt.Sprintf("%s: a reminder at %s that opens its page: /t/%s/%s.", name, when, records.ReminderType, rec.ID), change: &c}
}

// MeetingLength is how long a meeting runs, an hour when it does not say.
func MeetingLength(ev *store.Record) time.Duration {
	s, _ := ev.Fields["starts"].(string)
	e, _ := ev.Fields["ends"].(string)
	st, err1 := time.Parse(time.RFC3339, s)
	en, err2 := time.Parse(time.RFC3339, e)
	if err1 != nil || err2 != nil || !en.After(st) {
		return time.Hour
	}
	return en.Sub(st)
}

// nextStart is a meeting's start still to come, or its one start: a
// repeating meeting whose time is past moves on to the next, one that
// is on now keeps its start.
func nextStart(starts, repeat string, now time.Time, length time.Duration) string {
	at := starts
	for i := 0; repeat != "" && i < 1000; i++ {
		t, err := time.Parse(time.RFC3339, at)
		if err != nil || !t.Add(length).Before(now) {
			break
		}
		next, ok := when.Next(repeat, at, now)
		if !ok {
			break
		}
		at = next
	}
	return at
}

func whenRepeat(repeat string) string { return when.RepeatText(repeat) }

// RemindersAbout are the reminders, still to answer, about a page.
func RemindersAbout(st *store.Store, about string) []*store.Record {
	all, _ := st.List(records.ReminderType, store.ListOptions{})
	var out []*store.Record
	for _, r := range all {
		if r.Fields["about"] == about && r.Fields["state"] != "done" {
			out = append(out, r)
		}
	}
	return out
}
