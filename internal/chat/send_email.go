package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
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
	Words: []string{"email", "mail", "write to", "let them know", "tell them", "reply"},
	Doing: saying("Writing an email"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		e, err := s.emailOf(call.Args)
		if err != nil {
			return fail("%v", err), true
		}
		detail := fmt.Sprintf("From %s, about %q:\n\n%s", s.MailFrom(), e.Subject, e.Text)
		for _, c := range e.Choices {
			detail += fmt.Sprintf("\n\nWith a link, %s: pressing it would %s.", c.Label, c.what)
		}
		return s.ask(question{"Email " + e.To + " this?", detail, "Send it", "Don't send"},
			map[string]any{"tool": "send_email", "to": e.To, "subject": e.Subject, "text": e.Text, "choices": e.Choices}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult {
		e, err := s.emailOf(call.Args)
		if err != nil {
			return fail("%v", err)
		}
		if len(e.Choices) > 0 {
			lines, err := s.offer(e.To, e.Choices) // offers.go
			if err != nil {
				return fail("not sent: %v", err)
			}
			e.Text += "\n\n" + lines
		}
		if err := s.Mail(e.To, e.Subject, e.Text); err != nil {
			return fail("not sent: %v", err)
		}
		return toolResult{text: "Sent to " + e.To + " from " + s.MailFrom() + ".", change: &records.Change{Action: "emailed", Detail: trim.Line(e.To+": "+e.Subject, 120)}}
	},
	Tool: llm.Tool{Name: "send_email",
		Description: "Send an email from the person's own mailbox (the one connected on Email in) to one address: to write back to someone who booked or asked through the published page, whose name and email are in the request's asked_by, or to anyone the person names. Write it as the person, plainly. It is put to the person first, word for word. choices are links in the email that whoever gets it can press, each a change it makes here as the person agreed, such as confirming or moving a tentative event (update_record) or making one (create_record); the first pressed is made and the rest close, and being chosen can set off an action (proposal, by_action=offer, state=accepted). To write to whoever asks each time the person answers them, make an action when changed, what proposal, only by_action=request and state=accepted (or dismissed), kind message.",
		Schema: obj(map[string]any{
			"to":      map[string]any{"type": "string", "description": "One email address, with the name if known: Pat Lee <pat@example.com>."},
			"subject": map[string]any{"type": "string"},
			"text":    map[string]any{"type": "string", "description": "The email, as plain text; the choices' links are put after it."},
			"choices": map[string]any{"type": "array", "items": obj(map[string]any{
				"label":  map[string]any{"type": "string", "description": "What the link says: Confirm, Thursday at 3pm instead."},
				"tool":   map[string]any{"type": "string", "enum": []string{"create_record", "update_record"}},
				"type":   map[string]any{"type": "string"},
				"id":     map[string]any{"type": "string"},
				"fields": map[string]any{"type": "object"},
			}, "label", "tool", "type", "fields")},
		}, "to", "subject", "text")}}

type emailArgs struct {
	To      string   `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	Choices []choice `json:"choices"`
}

// emailOf reads a call: one address, a subject and the text, from a
// mailbox that is connected.
func (s *Service) emailOf(raw json.RawMessage) (emailArgs, error) {
	var e emailArgs
	json.Unmarshal(raw, &e)
	e.To, e.Subject, e.Text = strings.TrimSpace(e.To), trim.Line(e.Subject, 200), strings.TrimSpace(e.Text)
	switch {
	case s.Mail == nil || s.MailFrom == nil || s.MailFrom() == "":
		return e, fmt.Errorf("no mailbox is connected to send from: the person connects theirs on Email in")
	case !strings.Contains(e.To, "@") || strings.ContainsAny(e.To, ",;\r\n"):
		return e, fmt.Errorf("say one email address to send to")
	case e.Subject == "" || e.Text == "":
		return e, fmt.Errorf("say a subject and what the email says")
	}
	var err error
	e.Choices, err = s.choicesOf(e.Choices)
	return e, err
}
