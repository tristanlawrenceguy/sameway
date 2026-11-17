package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// An event goes out as a real invitation: whoever gets it answers with
// their own mail app's Yes, No or Maybe, and it is in their calendar. Its
// organiser is the address Sameway reads, so the answer comes back, and
// is kept on the event, told to the owner, and logged, only from an
// address it went to. Sent again after a change it moves there, with a
// higher sequence; cancelled, it is taken away.
func TestAnEventGoesOutAsAnInvitationAndTheAnswerComesBack(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	srv := h.(*server.Server)
	type mail struct{ to, text, calendar string }
	sent := make(chan mail, 3)
	a.Chat.MailFrom = func() string { return "tris+sameway@example.com" }
	a.Chat.Mail = func(m chat.Outgoing) (string, error) {
		sent <- mail{m.To, m.Text, strings.ReplaceAll(string(m.Calendar), "\r\n ", "")} // long lines unfolded
		return "<1@example.com>", nil
	}
	rang := make(chan string, 4)
	srv.OnRing(func(title, text, _ string) { rang <- title + ": " + text })
	ev, _ := a.Store.Create("event", map[string]any{"title": "Meeting with Pat Lee", "starts": "2026-03-13T14:00:00Z", "ends": "2026-03-13T15:00:00Z"})

	invite := func(extra string) mail {
		t.Helper()
		text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"Pat Lee <pat@example.com>","subject":"Friday","text":"See you Friday.","invite":"`+ev.ID+`"`+extra+`}`))
		ps := a.Chat.Proposals()
		if isErr || len(ps) == 0 {
			t.Fatalf("the invitation is put to the owner: %v %s", isErr, text)
		}
		wantStatus(t, postForm(t, h, "/proposal/"+ps[len(ps)-1].ID+"/accept", nil), http.StatusSeeOther)
		return <-sent
	}
	if text, isErr := a.Chat.Call("send_email", json.RawMessage(`{"to":"pat@example.com","subject":"x","text":"y","invite":"nothing"}`)); !isErr || !strings.Contains(text, "names no event") {
		t.Errorf("an invitation to nothing goes back to the assistant: %s", text)
	}
	if text, _ := a.Chat.Call("send_email", json.RawMessage(`{"to":"Pat Lee <pat@example.com>","subject":"Friday","text":"See you Friday.","invite":"`+ev.ID+`"}`)); !strings.Contains(text, "their mail app offers Yes, No and Maybe") {
		t.Errorf("the owner reads what the invitation does: %s", text)
	}
	a.Chat.Dismiss(a.Chat.Proposals()[0].ID)

	m := invite("")
	for _, want := range []string{"METHOD:REQUEST", "UID:" + ev.ID + "@sameway", "SUMMARY:Meeting with Pat Lee", "DTSTART:20260313T140000Z", "ORGANIZER:mailto:tris+sameway@example.com", "ATTENDEE;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE;CN=Pat Lee:mailto:pat@example.com", "SEQUENCE:1", "STATUS:CONFIRMED"} {
		if !strings.Contains(m.calendar, want) {
			t.Errorf("the invitation has %q:\n%s", want, m.calendar)
		}
	}
	if m.text != "See you Friday." || m.to != "Pat Lee <pat@example.com>" {
		t.Errorf("with the words, to them: %+v", m)
	}
	if e, _ := a.Store.Get("event", ev.ID); strings.Join(records.StringList(e.Fields["answers"]), "|") != "Pat Lee <pat@example.com>: invited" {
		t.Errorf("who it went to is kept on the event: %v", e.Fields["answers"])
	}

	reply := func(from, state string) []byte {
		return []byte("BEGIN:VCALENDAR\r\nMETHOD:REPLY\r\nBEGIN:VEVENT\r\nUID:" + ev.ID + "@sameway\r\nATTENDEE;PARTSTAT=" + state + ";CN=Someone:mailto:" + from + "\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")
	}
	a.Chat.InvitationAnswered(convert.Answers(reply("mallory@example.com", "DECLINED")))
	if e, _ := a.Store.Get("event", ev.ID); strings.Contains(strings.Join(records.StringList(e.Fields["answers"]), "|"), "declined") {
		t.Fatal("an answer from an address it did not go to is not theirs to give")
	}
	a.Chat.InvitationAnswered(convert.Answers(reply("pat@example.com", "ACCEPTED")))
	if e, _ := a.Store.Get("event", ev.ID); strings.Join(records.StringList(e.Fields["answers"]), "|") != "Pat Lee <pat@example.com>: accepted" {
		t.Errorf("their answer is kept, by the name it went to: %v", e.Fields["answers"])
	}
	if got := <-rang; !strings.Contains(got, `Pat Lee <pat@example.com> accepted your invitation to "Meeting with Pat Lee"`) {
		t.Errorf("the owner is told: %s", got)
	}

	a.Store.Update("event", ev.ID, map[string]any{"starts": "2026-03-13T16:00:00Z", "ends": "2026-03-13T17:00:00Z"})
	m = invite("")
	if !strings.Contains(m.calendar, "SEQUENCE:2") || !strings.Contains(m.calendar, "DTSTART:20260313T160000Z") {
		t.Errorf("sent again, it moves in their calendar: %s", m.calendar)
	}
	m = invite(`,"cancel":true`)
	if !strings.Contains(m.calendar, "METHOD:CANCEL") || !strings.Contains(m.calendar, "STATUS:CANCELLED") || !strings.Contains(m.calendar, "SEQUENCE:3") {
		t.Errorf("cancelled, it is taken away: %s", m.calendar)
	}
	if e, _ := a.Store.Get("event", ev.ID); strings.Join(records.StringList(e.Fields["answers"]), "|") != "Pat Lee <pat@example.com>: cancelled" {
		t.Errorf("and kept so: %v", e.Fields["answers"])
	}
}
