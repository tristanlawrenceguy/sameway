package convert

import (
	"strings"
	"testing"
	"time"
)

const calendar = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Google Inc//Google Calendar//EN\r\n" +
	"BEGIN:VEVENT\r\nUID:standup@example.com\r\nDTSTART;TZID=Europe/London:20261005T093000\r\nDTEND;TZID=Europe/London:20261005T094500\r\n" +
	"RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR\r\nSUMMARY:Stand-up\r\nLOCATION:https://meet.example.com/abc\r\n" +
	"DESCRIPTION:Three things:\\nwhat\\, why\\; when\r\n  and a folded line\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:bday\r\nDTSTART;VALUE=DATE:20261012\r\nSUMMARY:Hana's birthday\r\nRRULE:FREQ=YEARLY\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:dentist\r\nDTSTART:20261007T140000Z\r\nDTEND:20261007T143000Z\r\nSUMMARY:Dentist\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:outlook\r\nDTSTART;TZID=\"GMT Standard Time\":20261008T100000\r\nSUMMARY:Review\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:gone\r\nDTSTART:20261009T100000Z\r\nSUMMARY:Cancelled one\r\nSTATUS:CANCELLED\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:standup@example.com\r\nRECURRENCE-ID;TZID=Europe/London:20261007T093000\r\nDTSTART;TZID=Europe/London:20261007T100000\r\nSUMMARY:Stand-up (moved)\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

// A calendar file is read into its events: all-day ones by their day,
// times in their zone (Outlook's zone names too), repeats as rules,
// escapes and folded lines undone; cancelled ones and single changes to a
// repeating one are left out.
func TestACalendarIsReadIntoItsEvents(t *testing.T) {
	t.Parallel()
	events := ParseICS([]byte(calendar), time.UTC)
	if len(events) != 4 {
		t.Fatalf("four events, got %d: %+v", len(events), events)
	}
	s := events[0]
	if s.Title != "Stand-up" || s.Starts != "2026-10-05T09:30:00+01:00" || s.Ends != "2026-10-05T09:45:00+01:00" || s.Repeat != "FREQ=WEEKLY;BYDAY=MO,WE,FR" || s.Where != "https://meet.example.com/abc" || s.UID != "standup@example.com" {
		t.Errorf("a timed, repeating event in its zone: %+v", s)
	}
	if s.Notes != "Three things:\nwhat, why; when and a folded line" {
		t.Errorf("escapes and a folded line undone: %q", s.Notes)
	}
	if b := events[1]; b.Starts != "2026-10-12" || b.Repeat != "FREQ=YEARLY" || b.Title != "Hana's birthday" {
		t.Errorf("an all-day event by its day: %+v", b)
	}
	if d := events[2]; d.Starts != "2026-10-07T14:00:00Z" {
		t.Errorf("a time in UTC: %+v", d)
	}
	if o := events[3]; o.Starts != "2026-10-08T10:00:00+01:00" {
		t.Errorf("Outlook's GMT Standard Time is London: %+v", o)
	}
	md, err := icsMarkdown([]byte(calendar))
	if err != nil || !strings.Contains(md, "- **Hana's birthday**, Mon 12 Oct 2026 (repeats)") || strings.Contains(md, "Cancelled") {
		t.Errorf("as a person reads it: %q %v", md, err)
	}
	if Kind("work.ics") != "calendar" {
		t.Error("an .ics is a calendar")
	}
}

// Source and configuration are read as a code block in their language,
// one block even when they hold a fence; a .env is kept, not read; and
// subtitles are read as a transcript.
func TestCodeAndSubtitlesAreRead(t *testing.T) {
	t.Parallel()
	res, err := Read("tidy.py", []byte("def tidy():\n    \"\"\"```not a fence```\"\"\"\n    return 1\n"))
	if err != nil || res.Kind != "code" || !strings.HasPrefix(res.Markdown, "````python\ndef tidy():") || !strings.HasSuffix(res.Markdown, "\n````") {
		t.Errorf("a Python file as one python block: %q %v", res.Markdown, err)
	}
	if Kind("deploy.sh") != "code" || Kind("schema.sql") != "code" || Kind("app.tsx") != "code" {
		t.Error("scripts, queries and components are code")
	}
	if Kind(".env") == "code" || Builtin("prod.env") {
		t.Error("a .env holds secrets and is not read")
	}
	res, err = Read("talk.srt", []byte("1\n00:00:01,000 --> 00:00:03,000\nWelcome.\n\n2\n00:01:02,500 --> 00:01:04,000\nThe pond.\n"))
	if err != nil || res.Kind != "captions" || res.Markdown != "[0:01] Welcome.\n\n[1:02] The pond." {
		t.Errorf("subtitles as a transcript: %q %v", res.Markdown, err)
	}
}
