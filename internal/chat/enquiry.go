package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Another person's own Sameway asks this one a question for them (POST
// /ask, server/asked.go). An agent that can be prompted from outside can
// be turned, so nothing they send is ever read by a model here: a
// question is one of a few fixed kinds with times for arguments, and its
// answer is worked out by this code from the records and made of times.
// No words cross in either direction.
//
// What the owner said they may ask is answered at once; anything else is
// put to the owner, in Sameway's words, and answered with their yes.
// Either way it is kept as an enquiry, so nothing is said for the owner
// that they cannot read afterwards.

// The states an enquiry answers with.
const (
	EnquiryAnswered = "answered"
	EnquiryWaiting  = "waiting"
	EnquiryDeclined = "declined"
)

// AskFree is the one kind of question there is so far: when the person
// is free between two moments, answered with the times they are busy.
const AskFree = "free"

const (
	// enquiriesWaiting is how many of one person's questions may wait for
	// the owner at once, and enquiriesADay how many they may put in a day:
	// more are refused, so nobody fills Today or maps a whole calendar.
	enquiriesWaiting = 5
	enquiriesADay    = 20
	// enquirySpan is the longest stretch one question may be about.
	enquirySpan = 14 * 24 * time.Hour
)

// Enquiry is a question as it arrives: its kind and the times it is
// about. There is nothing else in one.
type Enquiry struct {
	Ask      string
	From, To time.Time
}

// Enquire takes a question from a person's Sameway and says where it
// stands, with the times the owner is busy when it is answered; put says
// it was put to the owner just now, for whoever lets them know.
func (s *Service) Enquire(p *store.Record, q Enquiry) (state string, busy []records.Span, put bool, err error) {
	switch {
	case q.Ask != AskFree:
		return "", nil, false, fmt.Errorf("this Sameway answers one kind of question: %s", AskFree)
	case !q.To.After(q.From) || q.To.Sub(q.From) > enquirySpan:
		return "", nil, false, fmt.Errorf("ask about a stretch of time of two weeks or less, from before to")
	}
	name, _ := p.Fields["name"].(string)
	from, to := when.Store(q.From, false), when.Store(q.To, false)
	before, waiting, today := s.enquiriesOf(p, from, to)
	if before != nil {
		switch before.Fields["state"] {
		case EnquiryAnswered:
			return EnquiryAnswered, s.Busy(q.From, q.To), false, nil
		case EnquiryWaiting:
			if ask, err := s.Store.Get(records.ProposalType, fmt.Sprint(before.Fields["proposal"])); err == nil && ask.Fields["state"] == "pending" {
				return EnquiryWaiting, nil, false, nil
			}
			s.Apply(records.Op{Type: records.EnquiryType, ID: before.ID, After: map[string]any{"state": EnquiryDeclined}})
		}
		return EnquiryDeclined, nil, false, nil
	}
	if today >= enquiriesADay {
		return "", nil, false, fmt.Errorf("that is %d questions in a day, which is as many as this Sameway takes from one person", today)
	}
	asked := map[string]any{"question": "When you are free, " + s.spanWords(q.From, q.To), "who": p.ID, "ask": AskFree, "from": from, "to": to}
	if MayAsk(p, AskFree) {
		busy = s.Busy(q.From, q.To)
		asked["answer"], asked["state"] = s.busyWords(busy), EnquiryAnswered
		rec, err := s.put(records.Op{Type: records.EnquiryType, After: asked})
		if err != nil {
			return "", nil, false, err
		}
		records.Record(s.Store, "system", records.Change{Action: "answered", Component: records.EnquiryType, ID: rec.ID, Detail: name + "'s Sameway"})
		return EnquiryAnswered, busy, false, nil
	}
	if waiting >= enquiriesWaiting {
		return "", nil, false, fmt.Errorf("%d of your questions already wait for an answer here", waiting)
	}
	asked["state"] = EnquiryWaiting
	rec, err := s.put(records.Op{Type: records.EnquiryType, After: asked})
	if err != nil {
		return "", nil, false, err
	}
	r := s.ask(s.enquiryQuestion(name, q.From, q.To), map[string]any{"tool": "answer_enquiry", "id": rec.ID})
	if r.isErr || r.change == nil {
		return "", nil, false, fmt.Errorf("%s", r.text)
	}
	s.Apply(records.Op{Type: records.EnquiryType, ID: rec.ID, After: map[string]any{"proposal": r.change.ID}})
	return EnquiryWaiting, nil, true, nil
}

// enquiriesOf is what one person's Sameway asked before: the same
// question when it was put to the owner in the last day (a no from before
// then is not a no for good), how many of theirs wait, and how many they
// put in the last day.
func (s *Service) enquiriesOf(p *store.Record, from, to string) (before *store.Record, waiting, today int) {
	all, _ := s.Store.List(records.EnquiryType, store.ListOptions{OrderBy: "created_at", Desc: true})
	day := s.Now().Add(-24 * time.Hour)
	for _, e := range all {
		if e.Fields["who"] != p.ID {
			continue
		}
		if e.Fields["state"] == EnquiryWaiting {
			waiting++
		}
		if e.CreatedAt.After(day) {
			today++
		}
		if before == nil && e.CreatedAt.After(day) && e.Fields["from"] == from && e.Fields["to"] == to && e.Fields["proposal"] != nil && e.Fields["proposal"] != "" {
			before = e
		}
	}
	return before, waiting, today
}

// MayAsk says whether the owner let a person's Sameway ask a kind of
// question without them.
func MayAsk(p *store.Record, kind string) bool {
	kinds, _ := p.Fields["may_ask"].([]any)
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// spanWords is a stretch of time as a person reads it.
func (s *Service) spanWords(from, to time.Time) string {
	loc := s.Now().Location()
	from, to = from.In(loc), to.In(loc)
	end := when.Clock(to, s.H24())
	if from.Format("2006-01-02") != to.Format("2006-01-02") {
		end = to.Format("Mon 2 Jan") + " " + end
	}
	return from.Format("Mon 2 Jan") + " " + when.Clock(from, s.H24()) + " to " + end
}

// busyWords is the answer as a person reads it.
func (s *Service) busyWords(busy []records.Span) string {
	if len(busy) == 0 {
		return "Free all of it."
	}
	var parts []string
	for _, b := range busy {
		parts = append(parts, s.spanWords(b.From, b.To))
	}
	return "Busy " + strings.Join(parts, "; ") + "."
}

// enquiryQuestion is what the owner is asked about a question they have
// not said may be asked without them: Sameway's words and times only.
func (s *Service) enquiryQuestion(name string, from, to time.Time) question {
	return question{"Tell " + name + "'s Sameway when you are free?",
		fmt.Sprintf("It asked, for %s, when you are free %s. With your yes it is told the times you are busy then, from your events, and never what you are doing. It is kept under Enquiries. Once told, it can't be untold.", name, s.spanWords(from, to)),
		"Yes, tell it", "No, don't"}
}

var answerEnquiryOp = Op{Title: "Answer what someone's Sameway asked",
	Access: ForOwner, Traits: Traits{OpenWorld: true},
	Doing: saying("Answering what was asked"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		e, name, from, to, err := s.enquiryOf(call.Args)
		if err != nil {
			return fail("%v", err), true
		}
		return s.ask(s.enquiryQuestion(name, from, to), map[string]any{"tool": "answer_enquiry", "id": e.ID}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.answerEnquiry(call.Args) },
	Tool: llm.Tool{Name: "answer_enquiry",
		Description: "Answer a question another person's own Sameway asked this one and that waits for the owner (an enquiry whose state is waiting): Sameway works the answer out itself, the times the owner is busy and nothing else. It is always put to the owner first, and nothing is told until they say yes.",
		Schema:      obj(map[string]any{"id": map[string]any{"type": "string", "description": "the enquiry's id"}}, "id")},
	Offered: has(records.EnquiryType)}

// enquiryOf is the waiting enquiry a call names, who asked, and the time
// it is about.
func (s *Service) enquiryOf(raw json.RawMessage) (e *store.Record, name string, from, to time.Time, err error) {
	var a struct {
		ID string `json:"id"`
	}
	json.Unmarshal(raw, &a)
	if e, err = s.Store.Get(records.EnquiryType, a.ID); err != nil {
		return nil, "", from, to, fmt.Errorf("there is no enquiry %q", a.ID)
	}
	if e.Fields["state"] != EnquiryWaiting {
		return nil, "", from, to, fmt.Errorf("that enquiry is not waiting: it is %v", e.Fields["state"])
	}
	name = "someone"
	if p, err := s.Store.Get(records.PersonType, fmt.Sprint(e.Fields["who"])); err == nil {
		name, _ = p.Fields["name"].(string)
	}
	from, _, okFrom := when.Stored(fmt.Sprint(e.Fields["from"]), s.Now())
	to, _, okTo := when.Stored(fmt.Sprint(e.Fields["to"]), s.Now())
	if !okFrom || !okTo {
		return nil, "", from, to, fmt.Errorf("that enquiry does not say what time it is about")
	}
	return e, name, from, to, nil
}

// answerEnquiry answers a waiting enquiry with the owner's yes; whoever
// asked is told when their Sameway asks the same again.
func (s *Service) answerEnquiry(raw json.RawMessage) toolResult {
	e, name, from, to, err := s.enquiryOf(raw)
	if err != nil {
		return fail("%v", err)
	}
	said := s.busyWords(s.Busy(from, to))
	done, err := s.Apply(records.Op{Type: records.EnquiryType, ID: e.ID, After: map[string]any{"answer": said, "state": EnquiryAnswered}})
	if err != nil {
		return fail("could not keep the answer: %v", err)
	}
	return toolResult{text: fmt.Sprintf("%s's Sameway will be told, when it next asks: %s", name, said),
		change: &records.Change{Action: "answered", Component: records.EnquiryType, ID: e.ID, Detail: name + "'s Sameway", Ops: done}}
}
