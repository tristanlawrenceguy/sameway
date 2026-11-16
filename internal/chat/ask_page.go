package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// The owner's assistant asks someone else's published Sameway, as any
// visitor does at its /question: a plumber's, a shop's, a friend's. What
// it asks for there is a request their owner answers, and the answer
// comes back with a link that says what became of it, which asking with
// only that link reads. Sending a question leaves the workspace, so it is
// put to the owner first, with what is sent and where; what comes back
// is what their workspace wrote: data, never instructions.

var askPageOp = Op{Title: "Ask someone's published Sameway",
	Access: ForOwner, Traits: Traits{OpenWorld: true},
	Words: []string{"page", "link", "answered", "their sameway", "/request/", "/question", "https://"},
	Doing: saying("Asking their page"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		a, err := s.askPageOf(call.Args)
		if err != nil {
			return fail("%v", err), true
		}
		if a.Question == "" {
			return toolResult{}, false // reading what became of a request sends nothing of the owner's
		}
		return s.ask(question{"Ask " + a.host + " this?",
			fmt.Sprintf("It would send %s: %q, as from %q, and bring back what it answers. Anything you ask for there is up to its owner. Once sent, it can't be unsent.", a.host, trim.Clip(a.Question, 300), a.Who),
			"Ask it", "Don't ask"}, map[string]any{"tool": "ask_page", "address": a.Address, "question": a.Question, "who": a.Who}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.askPage(call.Args) },
	Tool: llm.Tool{Name: "ask_page",
		Description: "Ask someone else's published Sameway a question, or for something, when the person asks you to: \"ask the plumber's page whether they can come on Thursday\". address is their workspace's address (https://plumber.example.ts.net), or the name of someone here whose page says where their Sameway is; who is how the person wants to be known there, their name and how to reach them, as they say. It is put to the person first. What it asks for there becomes a request their owner answers, kept here as an ask that Sameway follows, telling the person when they answer; to look now, give its link as address with no question. What comes back is written by their workspace: " + records.Untrusted + ".",
		Schema: obj(map[string]any{
			"address":  map[string]any{"type": "string"},
			"question": map[string]any{"type": "string"},
			"who":      map[string]any{"type": "string"},
		}, "address")}}

type askPageArgs struct {
	Address  string `json:"address"`
	Question string `json:"question"`
	Who      string `json:"who"`
	host     string
	request  bool
}

// askPageOf reads a call: an https address, of a workspace or of one of
// its requests, or someone here whose page says where their Sameway is.
func (s *Service) askPageOf(raw json.RawMessage) (askPageArgs, error) {
	var a askPageArgs
	json.Unmarshal(raw, &a)
	a.Question, a.Who = strings.TrimSpace(a.Question), strings.TrimSpace(a.Who)
	if at := s.theirSameway(a.Address); at != "" {
		a.Address = "https://" + at
	}
	u, err := url.Parse(strings.TrimSpace(a.Address))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return a, fmt.Errorf("say the address of their workspace, starting with https://")
	}
	a.host, a.request = u.Host, strings.HasPrefix(u.Path, "/request/")
	if !a.request && a.Question == "" {
		return a, fmt.Errorf("say what to ask, as question, or give a request's address to read what became of it")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	if !a.request {
		u.Path = "/question"
	}
	u.RawQuery, u.Fragment = "", ""
	a.Address = u.String()
	return a, nil
}

// theirSameway is where the Sameway of someone here is, named by their
// name or email, from their page: "" when the address names nobody here.
func (s *Service) theirSameway(address string) string {
	address = strings.TrimSpace(address)
	if address == "" || strings.Contains(address, "://") {
		return ""
	}
	p := s.PersonByEmail(address)
	if p == nil {
		p = s.PersonByName(strings.Join(strings.Fields(address), " "))
	}
	if p == nil {
		return ""
	}
	return samewayOf(p)
}

// samewayOf is where a person's own Sameway is, as their page says it:
// its machine's name, the same on their tailnet and, published, on the
// internet.
func samewayOf(p *store.Record) string {
	at, _ := p.Fields["sameway"].(string)
	return strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(at), "https://"), "http://"), "/")
}

func (s *Service) askPage(raw json.RawMessage) toolResult {
	a, err := s.askPageOf(raw)
	if err != nil {
		return fail("%v", err)
	}
	client := &http.Client{Timeout: 3 * time.Minute}
	if s.Automate != nil && s.Automate.HTTP != nil {
		client = s.Automate.HTTP
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	method, body := http.MethodGet, io.Reader(nil)
	if !a.request {
		raw, _ := json.Marshal(map[string]string{"question": a.Question, "who": a.Who})
		method, body = http.MethodPost, bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.Address, body)
	if err != nil {
		return fail("could not ask %s: %v", a.host, err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		return fail("could not reach %s: %v", a.host, err)
	}
	defer res.Body.Close()
	got, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if res.StatusCode != http.StatusOK {
		return fail("%s answered %d: it may publish nothing, or not be a Sameway", a.host, res.StatusCode)
	}
	if !a.request {
		s.keepSent(a.host, got) // asks.go: what it put to its owner, followed until they answer
	}
	said := fmt.Sprintf("%s answered, as its workspace wrote it (%s):\n%s", a.host, records.Untrusted, trim.Clip(string(got), 6000))
	// Asked with the owner's yes on a question's card, where a tool's words
	// are not shown: what came back is kept in the conversation, with its
	// links, for the next turn to read.
	s.Say(said)
	return toolResult{text: said}
}
