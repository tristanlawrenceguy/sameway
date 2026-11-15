package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// One person's Sameway asks another's. Each keeps their own workspace and
// their own assistant; a question goes between them over Tailscale, which
// says who is asking, and nobody opens anyone's workspace. The one asked
// decides what is answered: what they said the other may ask is answered
// at once, the rest waits for their yes (enquiry.go).
//
// What is sent is a kind of question and two times, and what comes back
// is a state and times: this code reads each as what it is and writes the
// words itself, so nothing the other side sent is ever read by the
// assistant here. Asking still leaves the workspace, so it is put to the
// owner first, like every other thing that cannot be taken back.

var askSomeoneOp = Op{Title: "Ask someone's Sameway when they are free",
	Access: ForOwner, Traits: Traits{OpenWorld: true},
	Words: []string{"ask ", "their sameway", "when is", "are we both", "both free"},
	Doing: saying("Asking their Sameway"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		a, p, at, err := s.askSomeoneOf(call.Args)
		if err != nil {
			return fail("%v", err), true
		}
		name, _ := p.Fields["name"].(string)
		return s.ask(question{"Ask " + name + "'s Sameway when " + name + " is free?",
			fmt.Sprintf("It would be asked, at %s, when %s is free %s. It answers with the times they are busy, or asks %s first. Once asked, it can't be unasked.", at, name, s.spanWords(a.from, a.to), name),
			"Ask it", "Don't ask"}, map[string]any{"tool": "ask_someone", "who": a.Who, "from": a.From, "to": a.To}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.askSomeone(call.Args) },
	Tool: llm.Tool{Name: "ask_someone",
		Description: "Ask another person's own Sameway when they are free, for the owner, when the owner asks to: \"ask Sam when we are both free on Thursday evening\". The person is one of the people here whose page says where their Sameway is. from and to are the stretch of time to ask about, two weeks at most, as a date and time (2026-11-19 17:00). Their Sameway answers with the times they are busy, never what they are doing, or asks them first, in which case ask the same again later. Asking is put to the owner first.",
		Schema: obj(map[string]any{
			"who":  map[string]any{"type": "string", "description": "their name, or the email on their page"},
			"from": map[string]any{"type": "string"},
			"to":   map[string]any{"type": "string"},
		}, "who", "from", "to")},
	Offered: has(records.PersonType)}

type askSomeoneArgs struct {
	Who      string `json:"who"`
	From     string `json:"from"`
	To       string `json:"to"`
	from, to time.Time
}

// askSomeoneOf reads a call: the person named, where their Sameway is,
// and the stretch of time, each checked before anything is asked.
func (s *Service) askSomeoneOf(raw json.RawMessage) (a askSomeoneArgs, p *store.Record, at string, err error) {
	json.Unmarshal(raw, &a)
	a.Who = strings.TrimSpace(a.Who)
	if p = s.PersonByEmail(a.Who); p == nil {
		p = s.PersonByName(strings.Join(strings.Fields(a.Who), " "))
	}
	if p == nil {
		return a, nil, "", fmt.Errorf("there is nobody called %q here: add them as a person first", a.Who)
	}
	name, _ := p.Fields["name"].(string)
	at, _ = p.Fields["sameway"].(string)
	at = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(at), "https://"), "http://"), "/")
	if at == "" || strings.ContainsAny(at, "/ ?#@") {
		return a, nil, "", fmt.Errorf("%s's page does not say where their Sameway is: put its Tailscale name there, such as home.tail1234.ts.net, as Their Sameway", name)
	}
	var okFrom, okTo bool
	a.from, _, okFrom = when.Parse(a.From, s.Now())
	a.to, _, okTo = when.Parse(a.To, s.Now())
	if !okFrom || !okTo || !a.to.After(a.from) || a.to.Sub(a.from) > enquirySpan {
		return a, nil, "", fmt.Errorf("say the stretch of time to ask about as from and to, each a date and time such as 2026-11-19 17:00, two weeks apart at most")
	}
	return a, p, at, nil
}

// theirAnswer is what the other Sameway sends back: a state and times.
type theirAnswer struct {
	State string         `json:"state"`
	Busy  []records.Span `json:"busy"`
	Error string         `json:"error"`
}

// askSomeone sends the question and says what came back, in this code's
// own words: of what was sent back, only the state and the times are used.
func (s *Service) askSomeone(raw json.RawMessage) toolResult {
	a, p, at, err := s.askSomeoneOf(raw)
	if err != nil {
		return fail("%v", err)
	}
	name, _ := p.Fields["name"].(string)
	var client *http.Client
	if s.Reach != nil {
		client = s.Reach()
	}
	if client == nil {
		return fail("this workspace is not on Tailscale, which is how it reaches %s's Sameway: ask me to put it on your phone, and I will walk you through it", name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"ask": AskFree, "from": a.from.UTC().Format(time.RFC3339), "to": a.to.UTC().Format(time.RFC3339)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+at+"/ask", bytes.NewReader(body))
	if err != nil {
		return fail("could not ask %s's Sameway: %v", name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return fail("could not reach %s's Sameway at %s", name, at)
	}
	defer res.Body.Close()
	var out theirAnswer
	if json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&out) != nil || res.StatusCode != http.StatusOK {
		// Why it refused is its words, so it is not passed on.
		return fail("%s's Sameway did not take the question (it answered %d). It takes questions only from people %s has named, a few a day.", name, res.StatusCode, name)
	}
	said := fmt.Sprintf("%s's Sameway may not say that on its own, so %s has been asked. Ask the same again later for the answer.", name, name)
	switch out.State {
	case EnquiryAnswered:
		var busy []records.Span
		for _, b := range out.Busy {
			if len(busy) < 100 && b.To.After(b.From) && !b.From.Before(a.from) && !b.To.After(a.to) {
				busy = append(busy, b)
			}
		}
		said = fmt.Sprintf("%s, %s: %s", name, s.spanWords(a.from, a.to), s.busyWords(busy))
	case EnquiryDeclined:
		said = name + "'s Sameway was given no answer to pass on."
	}
	// The owner said yes on a question's card, where a tool's words are
	// not shown: what came back is said in the conversation.
	s.Say(said)
	return toolResult{text: said, change: &records.Change{Action: "asked", Component: records.PersonType, ID: p.ID, Detail: name + "'s Sameway when they are free"}}
}

var letAskOp = Op{Title: "Let someone's Sameway ask yours",
	Access: ForOwner,
	Words:  []string{"may ask", "can ask", "let ", "their sameway"},
	Doing:  saying("Changing what they may ask"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		a := letAskOf(call.Args)
		if len(a.MayAsk) == 0 {
			return toolResult{}, false // taking it away is done at once
		}
		if !strings.Contains(a.Email, "@") {
			return fail("let_ask needs the email they sign in to Tailscale with, such as sam@example.com"), true
		}
		who := a.Email
		if a.Name != "" {
			who = a.Name + " (" + a.Email + ")"
		}
		return s.ask(question{"Let " + who + "'s Sameway ask yours when you are free?",
			fmt.Sprintf("Their own Sameway, signed in to Tailscale as %s, could ask this one when you are free, up to %d times a day, and be told the times you are busy without asking you: never what you are doing, and nothing else. It opens nothing here, and what it asks is kept under Enquiries. You can take it back at any time.", a.Email, enquiriesADay),
			"Yes, let it ask", "No"}, map[string]any{"tool": "let_ask", "email": a.Email, "name": a.Name, "may_ask": a.MayAsk}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.letAsk(letAskOf(call.Args)) },
	Tool: llm.Tool{Name: "let_ask",
		Description: "Say what another person's own Sameway may ask this one and be told without asking the owner, when the owner says so: \"Sam can ask when I'm free\". may_ask is the kinds of question: free is the times the owner is busy between two moments, never what they are doing; an empty list takes it away, at once. They are matched by the email they sign in to Tailscale with, and their Sameway reaches this one once the owner shares this machine with them in Tailscale (or they are on the same tailnet). It gives no access to the workspace. Giving it is put to the owner first.",
		Schema: obj(map[string]any{
			"email":   map[string]any{"type": "string"},
			"name":    map[string]any{"type": "string"},
			"may_ask": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{AskFree}}},
		}, "email", "may_ask")},
	Offered: has(records.PersonType, records.EnquiryType)}

// enquiryOps are the ops between one person's Sameway and another's.
var enquiryOps = []Op{askSomeoneOp, letAskOp, answerEnquiryOp}

type letAskArgs struct {
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	MayAsk []string `json:"may_ask"`
}

func letAskOf(raw json.RawMessage) letAskArgs {
	var a letAskArgs
	json.Unmarshal(raw, &a)
	a.Email = strings.ToLower(strings.TrimSpace(a.Email))
	return a
}

// letAsk sets what a person's Sameway may ask, making the person when
// there is none with that email yet.
func (s *Service) letAsk(a letAskArgs) toolResult {
	may := []any{}
	for _, k := range a.MayAsk {
		if k != AskFree {
			return fail("%q is not a kind of question: there is %s", k, AskFree)
		}
		may = []any{AskFree}
	}
	p := s.PersonByEmail(a.Email)
	if p == nil && len(may) == 0 {
		return toolResult{text: a.Email + " could ask nothing"}
	}
	op := records.Op{Type: records.PersonType, After: map[string]any{"may_ask": may}}
	name := a.Name
	if p == nil {
		if name == "" {
			name, _, _ = strings.Cut(a.Email, "@")
		}
		op.After = map[string]any{"name": name, "email": a.Email, "may_ask": may}
	} else {
		op.ID = p.ID
		name, _ = p.Fields["name"].(string)
	}
	done, err := s.Apply(op)
	if err != nil {
		return fail("could not change what they may ask: %v", err)
	}
	said, action := name+"'s Sameway can now ask when you are free", "let ask"
	if len(may) == 0 {
		said, action = name+"'s Sameway can no longer ask anything without you", "stopped asking by"
	}
	return toolResult{text: said, change: &records.Change{Action: action, Component: records.PersonType, ID: done[0].ID, Detail: name, Ops: done}}
}
