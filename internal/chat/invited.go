package chat

import (
	"bytes"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// An invitation someone emails the owner, from Gmail, Outlook or another
// Sameway, is a request like any: an event for the owner's calendar,
// from whoever sent it, waiting on Today for a yes, and nothing on the
// calendar before it. The answer goes back to them as their calendar
// takes one (export.Reply), which is how their Yes, No and Maybe learn
// it, and in the thread (email_conversation.go).

// Invited puts the invitations in a calendar that came by email from
// from, in the thread root, to the owner; me is the owner's address it
// came to, which the answer goes back as.
func (s *Service) Invited(data []byte, from, root, me string) bool {
	if !strings.Contains(strings.ToUpper(string(data)), "METHOD:REQUEST") || !s.hasTypes(records.EventType) {
		return false
	}
	them := s.For(records.Visitor{Access: records.Public, Name: from, Device: ByEmail, Login: "mail:" + root})
	for _, ev := range convert.ParseICS(data, s.Now().Location()) {
		if ev.UID == "" || s.EventByUID(ev.UID) != nil || ev.Starts == "" {
			continue // one already here is theirs to change, not asked again
		}
		fields := map[string]any{"title": trim.Line(ev.Title, 200), "starts": ev.Starts, "uid": ev.UID}
		if ev.Ends != "" {
			fields["ends"] = ev.Ends
		}
		if ev.Where != "" {
			fields["where"] = trim.Line(ev.Where, 300)
		}
		what := them.requestWords("create_record", records.EventType, "", map[string]any{"title": fields["title"], "starts": ev.Starts})
		if what == "" {
			continue
		}
		action := map[string]any{"tool": "create_record", "type": records.EventType, "fields": fields}
		summary, token, err := them.putRequest(action, strings.Replace(what, "make an event", "come to an event", 1), "An invitation by email")
		if err != nil {
			continue
		}
		if p := s.RequestByToken(token); p != nil {
			s.Apply(records.Op{Type: records.ProposalType, ID: p.ID, After: map[string]any{"reply_as": me}})
		}
		if s.Requested != nil {
			s.Requested(summary)
		}
	}
	return true
}

// ThreadOf is the conversation an email here is in, by its first email,
// found by its Message-ID, or "".
func (s *Service) ThreadOf(messageID string) string {
	want := strings.Trim(strings.TrimSpace(messageID), "<>")
	emails, _ := s.Store.List("email", store.ListOptions{})
	for _, e := range emails {
		if id, _ := e.Fields["message_id"].(string); id != "" && strings.Trim(id, "<>") == want {
			return rootOf(e)
		}
	}
	return ""
}

// EventByUID is the event a calendar knows by that id, or nil.
func (s *Service) EventByUID(uid string) *store.Record {
	recs, _ := s.Store.List(records.EventType, store.ListOptions{})
	for _, r := range recs {
		if r.Fields["uid"] == uid {
			return r
		}
	}
	return nil
}

// invitationReply is the answer to an invitation a request came from, as
// its organiser's calendar takes it, or nil for any other request.
func (s *Service) invitationReply(p *store.Record, organizer string, yes bool) []byte {
	me, _ := p.Fields["reply_as"].(string)
	action, _ := p.Fields["action"].(map[string]any)
	fields, _ := action["fields"].(map[string]any)
	t, ok := s.Store.Types().Get(records.EventType)
	if me == "" || fields["uid"] == nil || !ok {
		return nil
	}
	var b bytes.Buffer
	if export.Reply(&b, t, &store.Record{Type: records.EventType, Fields: fields}, organizer, me, yes) != nil {
		return nil
	}
	return b.Bytes()
}
