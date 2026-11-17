package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// The assistant writes to someone by email, from the owner's own mailbox,
// the one Email in reads: whoever booked through the published page, who
// asked a question there, or anyone the owner names. Every email is put
// to the owner first, word for word, and goes only with their yes, since
// it speaks for them and cannot be unsent. With an action that waits for
// something to happen, such as the owner answering a request, this is how
// anyone who asked hears back without Sameway of their own.

var sendEmailOp = Op{Title: "Send an email",
	Access: ForOwner, Traits: Traits{OpenWorld: true},
	Words: []string{"email", "mail", "write to", "let them know", "tell them", "reply", "invite", "invitation"},
	Doing: saying("Writing an email"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		e, err := s.emailOf(call.Args)
		if err == nil {
			err = s.canSend(e)
		}
		if err != nil {
			return fail("%v", err), true
		}
		detail := fmt.Sprintf("From your mailbox, about %q:\n\n%s", e.Subject, e.Text)
		for _, c := range e.Choices {
			detail += fmt.Sprintf("\n\nWith a link, %s: pressing it would %s.", c.Label, c.what)
		}
		if e.Invite != "" {
			detail += "\n\n" + s.inviteWords(e) // invitations.go
		}
		if len(e.people) > 1 {
			detail += "\n\nEach gets it on their own, and answers in their own thread."
		}
		return s.ask(question{"Email " + schema.AndNames(e.people) + " this?", detail, "Send it", "Don't send"},
			map[string]any{"tool": "send_email", "to": e.To, "subject": e.Subject, "text": e.Text, "choices": e.Choices, "invite": e.Invite, "cancel": e.Cancel}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult {
		e, err := s.emailOf(call.Args)
		if err == nil && (s.Mail == nil || s.MailFrom == nil || s.MailFrom() == "") {
			err = errNoMailbox
		}
		if err != nil {
			return fail("%v", err)
		}
		var went, notSent []string
		for _, to := range e.people {
			one := e
			one.To = to
			if err := s.sendOne(one); err != nil {
				notSent = append(notSent, to+" ("+err.Error()+")")
				continue
			}
			went = append(went, to)
		}
		if len(went) == 0 {
			return fail("not sent: %s", strings.Join(notSent, "; "))
		}
		said := "Sent to " + schema.AndNames(went) + " from " + s.MailFrom() + ", each on their own; their answers come back in their own threads."
		if len(notSent) > 0 {
			said += " Not sent to " + strings.Join(notSent, "; ") + "."
		}
		return toolResult{text: said, change: &records.Change{Action: "emailed", Detail: trim.Line(schema.AndNames(went)+": "+e.Subject, 120)}}
	},
	Tool: llm.Tool{Name: "send_email",
		Description: "Send an email from the person's own mailbox (the one connected on Email in) to one address: to write back to someone who booked or asked through the published page, whose name and email are in the request's asked_by, or to anyone the person names. Write it as the person, plainly. When they write back, Sameway answers them in the thread with their reach alone (free_times, and requests for the person's yes), as a conversation. It is put to the person first, word for word, so when they ask for an email or an invitation, call it rather than asking them in words first. choices are links in the email that whoever gets it can press, each a change it makes here as the person agreed, such as confirming or moving a tentative event (update_record) or making one (create_record); the first pressed is made and the rest close, and being chosen can set off an action (proposal, by_action=offer, state=accepted). invite sends an event as a real calendar invitation instead, for Yes, No and Maybe in their own mail app; an answer is logged on the event (when changed, what event), so an action can wait for it. To write to whoever asks each time the person answers them, make an action when changed, what proposal, only by_action=request and state=accepted (or dismissed), kind message.",
		Schema: obj(map[string]any{
			"to":      map[string]any{"type": "string", "description": "Who to send it to: one address, with the name if known (Pat Lee <pat@example.com>), or several separated by commas to ask around or arrange something with a group: each gets it on their own, in a thread of their own, answered as a conversation, and all wait on one yes."},
			"subject": map[string]any{"type": "string"},
			"text":    map[string]any{"type": "string", "description": "The email, as plain text; the choices' links are put after it."},
			"choices": map[string]any{"type": "array", "items": obj(map[string]any{
				"label":  map[string]any{"type": "string", "description": "What the link says: Confirm, Thursday at 3pm instead."},
				"tool":   map[string]any{"type": "string", "enum": []string{"create_record", "update_record"}},
				"type":   map[string]any{"type": "string"},
				"id":     map[string]any{"type": "string"},
				"fields": map[string]any{"type": "object"},
			}, "label", "tool", "type", "fields")},
			"invite": map[string]any{"type": "string", "description": "An event's id, to send it as a real invitation: their mail app shows Yes, No and Maybe and puts it in their calendar, and their answer comes back to the event's answers. Send it again after the event changes to move it in their calendar."},
			"cancel": map[string]any{"type": "boolean", "description": "With invite, take the event out of their calendar instead."},
		}, "to", "subject", "text")}}

type emailArgs struct {
	To      string   `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	Choices []choice `json:"choices"`
	Invite  string   `json:"invite"`
	Cancel  bool     `json:"cancel"`
	people  []string // who it goes to, each on their own
}

// plainAddress is an address as a person writes it, Pat Lee
// <pat@example.com>, quoted only where its name must be.
func plainAddress(a *mail.Address) string {
	switch {
	case a.Name == "":
		return a.Address
	case strings.ContainsAny(a.Name, `,;:"<>@()[]\`):
		return a.String()
	}
	return a.Name + " <" + a.Address + ">"
}

// mostPeople is how many one email goes to, each on their own.
const mostPeople = 20

// sendOne sends an email to one of those it goes to, with its choices and
// invitation made for them.
func (s *Service) sendOne(e emailArgs) error {
	if len(e.Choices) > 0 {
		lines, err := s.offer(e.To, e.Choices) // offers.go
		if err != nil {
			return err
		}
		e.Text += "\n\n" + lines
	}
	var calendar []byte
	if e.Invite != "" {
		var err error
		if calendar, err = s.invitation(e); err != nil { // invitations.go
			return err
		}
	}
	return s.sent(Outgoing{To: e.To, Subject: e.Subject, Text: e.Text, Calendar: calendar}, "") // email_conversation.go
}

var errNoMailbox = errors.New("no mailbox is connected to send from: the person connects theirs on Email in")

// canSend says, before an email is put to the person, what would stop it
// going, where this process knows: a tool run by Claude Code in a process
// of its own does not, and the email is sent from the server's.
func (s *Service) canSend(e emailArgs) error {
	if s.MailFrom != nil && s.MailFrom() == "" {
		return errNoMailbox
	}
	if len(e.Choices) > 0 && s.Address != nil && s.address() == "" {
		return errUnreachable
	}
	return nil
}

// emailOf reads a call: one address, a subject and the text, from a
// mailbox that is connected.
func (s *Service) emailOf(raw json.RawMessage) (emailArgs, error) {
	var e emailArgs
	json.Unmarshal(raw, &e)
	e.To, e.Subject, e.Text = strings.TrimSpace(e.To), trim.Line(e.Subject, 200), strings.TrimSpace(e.Text)
	list, err := mail.ParseAddressList(strings.ReplaceAll(e.To, ";", ","))
	switch {
	case err != nil || len(list) == 0 || strings.ContainsAny(e.To, "\r\n"):
		return e, fmt.Errorf("say who to send it to: an email address, or several separated by commas")
	case len(list) > mostPeople:
		return e, fmt.Errorf("that is %d people; one email goes to at most %d, each on their own", len(list), mostPeople)
	case e.Subject == "" || e.Text == "":
		return e, fmt.Errorf("say a subject and what the email says")
	}
	for _, a := range list {
		e.people = append(e.people, plainAddress(a))
	}
	if e.Invite != "" {
		if _, err := s.inviteOf(e.Invite); err != nil { // invitations.go
			return e, err
		}
	}
	e.Choices, err = s.choicesOf(e.Choices)
	return e, err
}
