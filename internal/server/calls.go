package server

import (
	"context"
	"encoding/xml"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/twilio"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Phone calls, as one more way in: the owner's Twilio number sends each
// call here, and the assistant talks with whoever rings as with a visitor
// to the published pages (chat/channel.go), by Twilio's own speech to
// text and its voice. What they ask to be made is a request, and the
// owner's answer goes back to them as a text message. A request to this
// address is only taken with Twilio's signature, made with the owner's
// token, so nobody else can pose as a call. An answer is worked out while
// the caller hears "one moment", since Twilio waits only seconds for one.

const (
	callsPrefix = "/calls/"
	callsPost   = "POST /calls/voice"
	twilioSID   = "twilio:sid"
	twilioToken = "twilio:token"
)

func (s *Server) twilio() (twilio.Account, bool) {
	a := twilio.Account{SID: s.keys().Get(twilioSID), Token: s.keys().Get(twilioToken), HTTP: s.app.Automate.HTTP}
	return a, a.SID != "" && a.Token != ""
}

// twilioConnected says whether calls are answered.
func (s *Server) twilioConnected() bool {
	_, ok := s.twilio()
	return ok
}

// call is one call's conversation, and the answer being worked out.
type call struct {
	said    []chat.Said
	working bool
	waits   int // turns heard in silence while it is worked out
	answer  string
}

type callsKey struct{}

type calls struct {
	mu  sync.Mutex
	all map[string]*call
}

func (s *Server) calls() *calls {
	return s.app.Store.Keep(callsKey{}, func() any { return &calls{all: map[string]*call{}} }).(*calls)
}

func (s *Server) callsPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	if _, ok := s.twilio(); ok {
		if at := s.internetAt(r); at != "" { // question.go
			b.WriteString(`<p>Connected to Twilio. In Twilio, under your number's voice settings, set "A call comes in" to this webhook, with POST:</p><p><code>` + template.HTMLEscapeString(at+"/calls/voice") + `</code></p><p>And to have texts answered too, under its messaging settings, set "A message comes in" to:</p><p><code>` + template.HTMLEscapeString(at+"/calls/text") + `</code></p><p>It is reached from the internet while something is published.</p>`)
		} else {
			b.WriteString(`<p>Connected to Twilio. Calls reach your assistant from the internet, so publish something first. The address to give Twilio is then here.</p>`)
		}
		b.WriteString(string(s.form(ui.Form{Action: "/calls/off", Button: &ui.Button{Label: "Stop answering calls", Variant: ui.Secondary}})))
		s.page(w, r, "Calls", template.HTML(b.String()), pageOptions{})
		return
	}
	b.WriteString(`<ol class="sw-stack"><li>Get a phone number from <a class="sw-link" href="https://www.twilio.com/console">Twilio</a>.</li><li>Copy your Account SID and Auth token from its console.</li><li>Paste them here.</li></ol>`)
	b.WriteString(string(s.form(ui.Form{Action: "/calls/connect", Class: "sw-stack",
		Body: s.part(ui.TextField{Label: "Account SID", Name: "sid", Required: true}) +
			s.part(ui.TextField{Label: "Auth token", Name: "token", Required: true, Hint: "It stays on this computer, in your keys."}),
		Button: &ui.Button{Label: "Connect"}})))
	s.page(w, r, "Calls", template.HTML(b.String()), pageOptions{Lede: "Your assistant answers your phone number."})
}

func (s *Server) callsConnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	acct := twilio.Account{SID: strings.TrimSpace(r.PostForm.Get("sid")), Token: strings.TrimSpace(r.PostForm.Get("token")), HTTP: s.app.Automate.HTTP}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	err := acct.Check(ctx)
	if err == nil {
		err = s.keys().Save(twilioSID, acct.SID)
	}
	if err == nil {
		err = s.keys().Save(twilioToken, acct.Token)
	}
	if err != nil {
		s.failed(w, r, "Not connected", err, "/calls")
		return
	}
	s.tellAt(w, r, outcome{Title: "Twilio connected", Text: "Set your number's voice webhook to the address on this page."}, "/calls")
}

func (s *Server) callsOff(w http.ResponseWriter, r *http.Request) {
	s.keys().Save(twilioSID, "")
	s.keys().Save(twilioToken, "")
	s.tellAt(w, r, outcome{Title: "Calls no longer answered", Text: "Your number is Twilio's to answer again."}, "/calls")
}

// voice answers one step of a call, as TwiML.
func (s *Server) voice(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	acct, ok := s.twilio()
	if !ok || !acct.Signed("https://"+r.Host+r.URL.RequestURI(), r.PostForm, r.Header.Get("X-Twilio-Signature")) {
		writeJSON(w, http.StatusForbidden, map[string]any{"said": "not a call: only Twilio, signed with the owner's token, is taken here"})
		return
	}
	sid, heard := r.PostForm.Get("CallSid"), strings.TrimSpace(r.PostForm.Get("SpeechResult"))
	cs := s.calls()
	cs.mu.Lock()
	c := cs.all[sid]
	if c == nil {
		c = &call{}
		cs.all[sid] = c
	}
	switch {
	case heard != "" && !c.working:
		c.said = append(c.said, chat.Said{Text: heard})
		c.working, c.answer = true, ""
		v := records.Visitor{Access: records.Public, Name: "A caller from " + r.PostForm.Get("From"), Device: "phone", Login: "call:" + r.PostForm.Get("From") + "|" + r.PostForm.Get("To")}
		said := append([]chat.Said(nil), c.said...)
		go s.answerCall(cs, c, v, said)
		cs.mu.Unlock()
		twiml(w, `<Say>One moment.</Say><Redirect method="POST">/calls/voice?wait=1</Redirect>`)
	case c.working:
		// Long silence on a phone sounds like a dropped call, so every
		// few seconds the caller hears it is still being worked on.
		c.waits++
		still := ""
		if c.waits%4 == 0 {
			still = `<Say>Still working on it.</Say>`
		}
		cs.mu.Unlock()
		twiml(w, still+`<Pause length="2"/><Redirect method="POST">/calls/voice?wait=1</Redirect>`)
	case c.answer != "":
		answer := c.answer
		c.answer, c.waits = "", 0
		cs.mu.Unlock()
		twiml(w, listen(answer, "Here it is again. "+answer))
	default:
		cs.mu.Unlock()
		twiml(w, listen("Hello, you have reached "+s.app.Workspace.Config.Name+"'s assistant. How can I help?", "I did not hear anything. How can I help?"))
	}
}

// listen says something and listens for the caller's answer. A caller
// who says nothing hears it once more, for whoever needs longer or did
// not catch it, before goodbye.
func listen(say, again string) string {
	return gather(callsPrefix+"voice", say) + gather(callsPrefix+"voice", again) + `<Say>Goodbye.</Say>`
}

// answerCall works out what to say to a caller, as a visitor's answer.
func (s *Server) answerCall(cs *calls, c *call, v records.Visitor, said []chat.Said) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	answer, err := s.app.Chat.For(v).AnswerChat(ctx, "a phone call, answered aloud: a few short spoken sentences, no lists or links. What they said was heard by speech recognition and may be wrong: say back what you understood (a day and time in words), so they can put it right, and ask them to say it again when it makes no sense", said)
	if err != nil || strings.TrimSpace(answer) == "" {
		answer = "Sorry, I could not answer just now."
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c.said = append(c.said, chat.Said{Ours: true, Text: answer})
	c.working, c.answer = false, answer
}

// gather says something and listens for what the caller says back, which
// Twilio posts to action. It
// waits 8 seconds for them to start and 3 seconds of quiet for them to
// finish: Twilio's "auto" ends at the first pause, which cuts off anyone
// who speaks slowly, stammers or stops to think.
func gather(action, say string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(say))
	return `<Gather input="speech" action="` + action + `" method="POST" timeout="8" speechTimeout="3"><Say>` + b.String() + `</Say></Gather>`
}

func twiml(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Response>` + body + `</Response>`))
}

// callTell tells a caller what the owner answered, by text message from
// the number they rang.
func (s *Server) callTell(v records.Visitor, text string) bool {
	from, to, ok := strings.Cut(strings.TrimPrefix(v.Login, "call:"), "|")
	acct, connected := s.twilio()
	if v.Device != "phone" || !ok || !connected {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return acct.Text(ctx, to, from, text) == nil
}
