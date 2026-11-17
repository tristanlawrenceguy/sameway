package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// An invitation someone emails the owner is a request like any: it waits
// on Today, nothing is on the calendar before the owner's yes, and the
// answer goes back to them as their calendar takes one, accepted or
// declined, in the thread; the same invitation again is not asked twice.
func TestAnInvitationEmailedToTheOwnerWaitsForTheirYes(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	var sent []chat.Outgoing
	a.Chat.MailFrom = func() string { return "tris+sameway@example.com" }
	a.Chat.Mail = func(m chat.Outgoing) (string, error) { sent = append(sent, m); return "<re@example.com>", nil }
	invite := func(uid, title, msgID string) string {
		a.Store.Create("email", map[string]any{"subject": "Invitation: " + title, "from": "Pat Lee <pat@example.com>", "body": "Pat has invited you.", "message_id": msgID, "from_me": false})
		data := "BEGIN:VCALENDAR\r\nMETHOD:REQUEST\r\nBEGIN:VEVENT\r\nUID:" + uid + "\r\nSUMMARY:" + title + "\r\nDTSTART:20260317T140000Z\r\nDTEND:20260317T150000Z\r\n" +
			"ORGANIZER;CN=Pat Lee:mailto:pat@example.com\r\nATTENDEE;PARTSTAT=NEEDS-ACTION;RSVP=TRUE:mailto:tris@example.com\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
		if !a.Chat.Invited([]byte(data), "Pat Lee <pat@example.com>", a.Chat.ThreadOf(msgID), "tris@example.com") {
			t.Fatal("an invitation is taken as one")
		}
		for _, p := range a.Chat.Suggestions() {
			if strings.Contains(p.Fields["summary"].(string), title) {
				return p.ID
			}
		}
		return ""
	}
	review := invite("abc@google.com", "Quarterly review", "<inv1@example.com>")
	if review == "" {
		t.Fatal("it waits on Today")
	}
	if p, _ := a.Store.Get("proposal", review); !strings.Contains(p.Fields["summary"].(string), `Pat Lee <pat@example.com> asks you to come to an event: "Quarterly review"`) {
		t.Errorf("in Sameway's words: %v", p.Fields["summary"])
	}
	if n, _ := a.Store.Count("event"); n != 0 {
		t.Fatal("nothing is on the calendar before the owner's yes")
	}
	wantStatus(t, postForm(t, h, "/suggested/yes", url.Values{"id": {review}}), http.StatusSeeOther)
	if ev := a.Chat.EventByUID("abc@google.com"); ev == nil || ev.Fields["title"] != "Quarterly review" {
		t.Fatal("with the yes it is on the calendar, by their id for it")
	}
	reply := strings.ReplaceAll(string(sent[0].Calendar), "\r\n ", "")
	for _, want := range []string{"METHOD:REPLY", "UID:abc@google.com", "ORGANIZER;CN=Pat Lee:mailto:pat@example.com", "ATTENDEE;PARTSTAT=ACCEPTED:mailto:tris@example.com"} {
		if !strings.Contains(reply, want) {
			t.Errorf("the answer goes back as their calendar takes it, %q:\n%s", want, reply)
		}
	}
	if sent[0].To != "Pat Lee <pat@example.com>" || sent[0].Refs[0] != "<inv1@example.com>" || !strings.Contains(sent[0].Text, "said yes to your invitation") {
		t.Errorf("to them, in the thread: %+v", sent[0])
	}

	if again := invite("abc@google.com", "Quarterly review", "<inv2@example.com>"); again != "" {
		t.Error("the same invitation again is not asked twice")
	}
	lunch := invite("lunch@google.com", "Team lunch", "<inv3@example.com>")
	wantStatus(t, postForm(t, h, "/suggested/no", url.Values{"id": {lunch}}), http.StatusSeeOther)
	if last := sent[len(sent)-1]; !strings.Contains(string(last.Calendar), "PARTSTAT=DECLINED") || a.Chat.EventByUID("lunch@google.com") != nil {
		t.Errorf("a no goes back declined, and nothing is made: %s", last.Calendar)
	}
}
