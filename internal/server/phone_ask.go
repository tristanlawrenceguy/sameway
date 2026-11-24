package server

import (
	"context"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Phoning a business to ask one question the owner agreed to (chat's
// phone_to_ask): Twilio rings the number from the owner's, and asks the
// address made for this one call, by a token nobody could guess, what to
// say. What it says is fixed here: who is calling and why, the question,
// once more if nothing was heard, and thank you. As an incoming call, only
// Twilio's signature, made with the owner's token, is taken there. What
// they answer goes to the owner as their words (chat.Phoned), and to no
// model.

const (
	phoneAskPrefix = callsPrefix + "ask/"
	phoneAskPost   = "POST /calls/ask/{token}"
)

// outCall is one call out: what it asks, and whether the owner was told.
type outCall struct {
	who, question string
	told          bool
}

type outCallsKey struct{}

type outCalls struct {
	mu  sync.Mutex
	all map[string]*outCall
}

func (s *Server) outCalls() *outCalls {
	return s.app.Store.Keep(outCallsKey{}, func() any { return &outCalls{all: map[string]*outCall{}} }).(*outCalls)
}

// phone rings number to ask who the question.
func (s *Server) phone(number, who, question string) error {
	acct, ok := s.twilio()
	at := ""
	if s.app.Chat.Address != nil {
		at = strings.TrimSuffix(s.app.Chat.Address(), "/")
	}
	switch {
	case !ok:
		return errors.New("Twilio is not connected: the person connects it on Calls")
	case at == "":
		return errors.New("Twilio needs this Sameway reachable from the internet to hear the answer: publish something first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	from, err := acct.Number(ctx)
	if err != nil {
		return err
	}
	token, oc := rand.Text(), s.outCalls()
	oc.mu.Lock()
	oc.all[token] = &outCall{who: who, question: question}
	oc.mu.Unlock()
	address := at + phoneAskPrefix + token
	if err := acct.Dial(ctx, from, number, address, address+"?ended=1"); err != nil {
		oc.mu.Lock()
		delete(oc.all, token)
		oc.mu.Unlock()
		return err
	}
	return nil
}

// phoneAsk answers one step of a call out, as TwiML: the script when it
// is picked up, thank you once they answer, and how it ended.
func (s *Server) phoneAsk(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	acct, ok := s.twilio()
	if !ok || !acct.Signed("https://"+r.Host+r.URL.RequestURI(), r.PostForm, r.Header.Get("X-Twilio-Signature")) {
		writeJSON(w, http.StatusForbidden, map[string]any{"said": "not a call: only Twilio, signed with the owner's token, is taken here"})
		return
	}
	token, heard := r.PathValue("token"), strings.TrimSpace(r.PostForm.Get("SpeechResult"))
	oc := s.outCalls()
	oc.mu.Lock()
	c := oc.all[token]
	ended := r.URL.Query().Has("ended")
	tell := c != nil && !c.told && (heard != "" || ended)
	if tell {
		c.told = true
	}
	if ended {
		delete(oc.all, token)
	}
	oc.mu.Unlock()
	if tell {
		s.app.Chat.Phoned(c.who, c.question, heard, r.PostForm.Get("CallStatus") == "completed")
	}
	switch {
	case c == nil || ended:
		twiml(w, `<Hangup/>`)
	case heard != "":
		twiml(w, `<Say>Thank you. Goodbye.</Say><Hangup/>`)
	default:
		var hello strings.Builder
		xml.EscapeText(&hello, []byte("Hello, this is an AI assistant calling for "+s.app.Workspace.Config.Name+", with one question. This call is transcribed."))
		action := phoneAskPrefix + token
		twiml(w, `<Say>`+hello.String()+`</Say>`+gather(action, c.question)+gather(action, c.question)+`<Say>Thank you. Goodbye.</Say><Hangup/>`)
	}
}
