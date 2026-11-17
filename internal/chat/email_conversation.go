package chat

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A conversation by email, the way meetings used to be arranged: the
// owner's assistant writes to someone (send_email, with the owner's yes),
// and when they write back, in their own words, the assistant answers in
// the same thread as it answers a visitor to the published pages, with
// their reach and nothing more: it reads what is published and when the
// owner is free, and what they ask to be made, such as a time to meet, is
// a request on Today. The owner's yes is told back to them in the thread,
// a time to meet as a real invitation. Every email Sameway sends asks for
// answers at the address it reads, which is how it knows them.

// ByEmail marks someone Sameway speaks with by email.
const ByEmail = "email"

// Outgoing is an email Sameway sends; the server sends it (mailin).
type Outgoing struct {
	To, Subject, Text string
	Calendar          []byte   // an invitation it carries, or nil
	Refs              []string // the thread it answers, oldest first
}

// sent sends an email from the owner's mailbox and keeps it as theirs,
// in its thread, root being the email the thread began with or "".
func (s *Service) sent(m Outgoing, root string) error {
	if s.Mail == nil || s.MailFrom == nil || s.MailFrom() == "" {
		return errNoMailbox
	}
	id, err := s.Mail(m)
	if err != nil {
		return err
	}
	if _, ok := s.Store.Types().Get("email"); !ok || id == "" {
		return nil
	}
	fields := map[string]any{"subject": trim.Line(m.Subject, 300), "from": s.MailFrom(), "received": when.Store(s.Now(), false),
		"body": m.Text, "message_id": id, "from_me": true, "tags": []any{}, "turn": records.Turn(true, m.Text)}
	ops := []records.Op{}
	if root != "" {
		fields["thread"] = root
		for _, e := range s.thread(root) { // the conversation's state is its latest
			if t, _ := e.Fields["turn"].(string); t != "" {
				ops = append(ops, records.Op{Type: "email", ID: e.ID, After: map[string]any{"turn": ""}})
			}
		}
	}
	_, err = s.Apply(append(ops, records.Op{Type: "email", After: fields})...)
	return err
}

// Conversation is the thread an email answers, by the emails it names,
// when Sameway sent one of them: "" for any other email.
func (s *Service) Conversation(refs []string) string {
	emails, _ := s.Store.List("email", store.ListOptions{OrderBy: "created_at"})
	for _, r := range refs {
		r = strings.Trim(strings.TrimSpace(r), "<>")
		for _, e := range emails {
			id, _ := e.Fields["message_id"].(string)
			if mine, _ := e.Fields["from_me"].(bool); mine && id != "" && strings.Trim(id, "<>") == r {
				if root, _ := e.Fields["thread"].(string); root != "" {
					return root
				}
				return e.ID
			}
		}
	}
	return ""
}

// letInBy is the person here an address belongs to, when they are let in.
func (s *Service) letInBy(from string) *store.Record {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return nil
	}
	p := s.PersonByEmail(addr.Address)
	if p == nil {
		return nil
	}
	switch p.Fields["access"] {
	case records.View, records.Edit, records.Host:
		return p
	}
	return nil
}

// Answers says which thread an email that came in is answered in, by its
// Message-ID and whom it is from: one Sameway began, or any from someone
// let in here whose mail server vouched for them; "" for none.
func (s *Service) Answers(messageID string, refs []string, from string, verified bool) string {
	if root := s.Conversation(refs); root != "" {
		return root
	}
	if !verified || s.letInBy(from) == nil {
		return ""
	}
	return s.ThreadOf(messageID) // invited.go
}

// thread is a conversation's emails, oldest first.
func (s *Service) thread(root string) []*store.Record {
	emails, _ := s.Store.List("email", store.ListOptions{OrderBy: "created_at"})
	var out []*store.Record
	for _, e := range emails {
		if e.ID == root || e.Fields["thread"] == root {
			out = append(out, e)
		}
	}
	return out
}

// AnswerMail answers the latest email of a conversation, as whoever wrote
// it, with their reach, and sends the answer in the thread. verified is
// whether their mail server vouched for who sent it: only then is someone
// let in here answered as themselves, with what they are let in to;
// anyone else has a stranger's reach.
func (s *Service) AnswerMail(ctx context.Context, root string, verified bool) error {
	mails := s.thread(root)
	if len(mails) == 0 {
		return nil
	}
	last := mails[len(mails)-1]
	if mine, _ := last.Fields["from_me"].(bool); mine {
		return nil
	}
	from, _ := last.Fields["from"].(string)
	v := records.Visitor{Access: records.Public, Name: from, Device: ByEmail, Login: "mail:" + root}
	if p := s.letInBy(from); p != nil && verified {
		v.Person, v.Access = p.ID, fmt.Sprint(p.Fields["access"])
	}
	them := s.For(v)
	if them.Provider == nil {
		return fmt.Errorf("no assistant to answer %s", from)
	}
	owner := s.Owner.Name
	if owner == "" {
		owner = "the owner"
	}
	system := fmt.Sprintf("You write for %s, by email, to %s, who is writing back in a conversation %s began. You hold their reach, nothing more: you read what they may read here (what %s published, and for someone %s let in, what they are let in to), and free_times says when they are free. Write the next email in the thread: short, plain, warm, as %s's assistant, signed as that. When they ask for something to be made, such as a time to meet (an event, with when it starts and ends and their name in the title), make it with create_record: it is not made but put to %s, who says yes or no, and they will hear back in this thread; say so, and never say it is booked. Search before saying you cannot answer. When what you read does not answer it, say so, and that %s sees this thread too; never say you did anything you did not do with a tool. When what they wrote is unclear, ask. Write only the email's words. %s",
		owner, from, owner, owner, owner, owner, owner, owner, records.DataNotInstructions)
	var msgs []llm.Message
	for _, m := range mails {
		body, _ := m.Fields["body"].(string)
		role := llm.RoleUser
		if mine, _ := m.Fields["from_me"].(bool); mine {
			role = llm.RoleAssistant
		}
		if n := len(msgs); n > 0 && msgs[n-1].Role == role {
			msgs[n-1].Content += "\n\n" + body
			continue
		}
		msgs = append(msgs, llm.Message{Role: role, Content: body})
	}
	if msgs[0].Role == llm.RoleAssistant {
		msgs = append([]llm.Message{{Role: llm.RoleUser, Content: "(" + owner + " wrote first.)"}}, msgs...)
	}
	defer them.tellOwner(s.Now())
	answer, err := them.answerWith(ctx, system, msgs)
	if err != nil || strings.TrimSpace(answer) == "" {
		return err
	}
	return s.sent(Outgoing{To: from, Subject: reSubject(last), Text: answer, Refs: threadRefs(mails)}, root)
}

// reSubject is the subject of an answer to an email.
func reSubject(e *store.Record) string {
	subject, _ := e.Fields["subject"].(string)
	if strings.HasPrefix(strings.ToLower(subject), "re:") {
		return subject
	}
	return "Re: " + subject
}

// threadRefs are the names of a thread's emails, oldest first.
func threadRefs(mails []*store.Record) []string {
	var out []string
	for _, m := range mails {
		if id, _ := m.Fields["message_id"].(string); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// tellByMail tells someone who asked by email what the owner answered,
// in their thread: a time to meet said yes to goes as a real invitation.
func (s *Service) tellByMail(p *store.Record, v records.Visitor, yes bool) {
	if now, err := s.Store.Get(records.ProposalType, p.ID); err == nil {
		p = now // with what the yes made
	}
	root := strings.TrimPrefix(v.Login, "mail:")
	mails := s.thread(root)
	if len(mails) == 0 {
		return
	}
	what, _ := p.Fields["asked"].(string)
	owner := s.Owner.Name
	if owner == "" {
		owner = "They"
	}
	m := Outgoing{To: v.Name, Subject: reSubject(mails[len(mails)-1]), Refs: threadRefs(mails),
		Text: owner + " said no to your request to " + what + ": nothing was made."}
	if reply := s.invitationReply(p, v.Name, yes); reply != nil { // invited.go: as their calendar answers
		m.Calendar = reply
		m.Text = owner + " said no to your invitation."
		if yes {
			m.Text = owner + " said yes to your invitation."
		}
	} else if yes {
		m.Text = owner + " said yes to your request to " + what + ": it is made."
		if typ, id, ok := strings.Cut(fmt.Sprint(p.Fields["made"]), "/"); ok && typ == records.EventType {
			if cal, err := s.invitation(emailArgs{To: v.Name, Invite: id}); err == nil { // invitations.go
				m.Calendar = cal
				m.Text += " It is in the invitation with this email, for your calendar."
			}
		}
	}
	s.sent(m, root)
}
