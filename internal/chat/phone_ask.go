package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// The assistant phones a business, from the owner's Twilio number, to
// ask one question the owner agreed to word for word: are you open on
// Sunday, do you have it in stock. The call follows a fixed script, said
// by the server (server/phone_ask.go): it is an AI calling for the owner,
// the question, and thank you. There is no conversation beyond it, and
// what they answer is kept and told to the owner as their words, never
// read by a model with tools. The owner's yes and the script are the
// safeguard: nothing is booked, paid, shared or agreed on a call.

var phoneAskOp = Op{Title: "Phone a business with one question",
	Access: ForOwner, Traits: Traits{OpenWorld: true},
	Doing: saying("Phoning"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		p, err := phoneOf(call.Args)
		if err == nil && (s.Phones == nil || !s.Phones()) {
			err = errNoTwilio
		}
		if err != nil {
			return fail("%v", err), true
		}
		return s.ask(question{fmt.Sprintf("Phone %s on %s?", p.Who, p.Number),
			fmt.Sprintf("Sameway would ring %s from your Twilio number and ask only this, word for word:\n\n\"%s\"\n\nIt says it is an AI assistant calling for you. It saves what they say and tells you. It books, pays, shares and agrees to nothing. A call can't be taken back.", p.Who, p.Question),
			"Phone " + p.Number, "Don't phone"},
			map[string]any{"tool": "phone_to_ask", "number": p.Number, "who": p.Who, "question": p.Question}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult {
		p, err := phoneOf(call.Args)
		if err == nil && s.Phone == nil {
			err = errNoTwilio
		}
		if err == nil {
			err = s.Phone(p.Number, p.Who, p.Question)
		}
		if err != nil {
			return fail("not phoned: %v", err)
		}
		return toolResult{text: "Ringing " + p.Who + " on " + p.Number + ". What they answer is told to the person in this chat and as a notification; nothing to do until then.",
			change: &records.Change{Action: "phoned", Detail: trim.Line(p.Who+": "+p.Question, 120)}}
	},
	Offered: func(s *Service, _ *llm.Tool) bool { return s.Phones != nil && s.Phones() },
	Tool: llm.Tool{Name: "phone_to_ask",
		Description: "Phone a business from the person's Twilio number to ask it one question, put to the person word for word first (so call it rather than asking in words). The call says it is an AI calling for them and asks only that; the answer is told to the person when it comes.",
		Schema: obj(map[string]any{
			"number":   map[string]any{"type": "string", "description": "The business's number in international form: +441234567890."},
			"who":      map[string]any{"type": "string", "description": "The business's name."},
			"question": map[string]any{"type": "string", "description": "One question, as it is to be said aloud."},
		}, "number", "who", "question")}}

var errNoTwilio = errors.New("Twilio is not connected: the person connects it on Calls")

type phoneArgs struct {
	Number   string `json:"number"`
	Who      string `json:"who"`
	Question string `json:"question"`
}

// e164 is a phone number as Twilio takes it: a plus, a country code and
// at most fifteen digits in all.
var e164 = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

func phoneOf(raw json.RawMessage) (phoneArgs, error) {
	var p phoneArgs
	json.Unmarshal(raw, &p)
	p.Number, p.Who, p.Question = trim.Line(p.Number, 20), trim.Line(p.Who, 100), trim.Flat(p.Question, 300)
	switch {
	case !e164.MatchString(p.Number):
		return p, fmt.Errorf("number %q is not in international form: a plus, the country code and the number, digits only, such as +441234567890", p.Number)
	case p.Who == "" || p.Question == "":
		return p, fmt.Errorf("say who is phoned and the one question to ask")
	}
	return p, nil
}

// Phoned tells the owner what a business answered, or that nobody did,
// in their chat and as a notification, and keeps it in the log. heard is
// their words as Twilio's speech to text gave them; picked is whether
// the call was answered at all.
func (s *Service) Phoned(who, question, heard string, picked bool) {
	said := fmt.Sprintf("%s answered \"%s\": \"%s\".", who, question, trim.Flat(heard, 1000))
	switch {
	case heard == "" && picked:
		said = fmt.Sprintf("%s picked up, but nothing was heard when asked \"%s\".", who, question)
	case heard == "":
		said = fmt.Sprintf("%s did not answer the phone, so \"%s\" was not asked.", who, question)
	}
	records.Record(s.Store, "system", records.Change{Action: "heard on the phone", Detail: trim.Line(said, 300)})
	s.tell("Phone answer", said, "/today") // email_conversation.go
}
