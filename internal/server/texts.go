package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Text messages to the owner's Twilio number, the one calls ring: whoever
// texts is answered as a visitor to the published pages, by text, in a
// conversation kept for each number, and what they ask to be made waits
// for the owner, whose answer comes back by text as a caller's does. As a
// call, it is taken only with Twilio's signature. Nobody is the owner by
// the number they text from, since a sender's number can be made up.

const (
	textsPost = "POST /calls/text"
	textsKept = 20 // the messages of a conversation kept, theirs and ours
)

// text takes one message: Twilio is answered at once, and the answer is
// sent when it is worked out, since Twilio waits only seconds.
func (s *Server) text(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	acct, ok := s.twilio()
	if !ok || !acct.Signed("https://"+r.Host+r.URL.RequestURI(), r.PostForm, r.Header.Get("X-Twilio-Signature")) {
		writeJSON(w, http.StatusForbidden, map[string]any{"said": "not a text: only Twilio, signed with the owner's token, is taken here"})
		return
	}
	from, to, body := r.PostForm.Get("From"), r.PostForm.Get("To"), strings.TrimSpace(r.PostForm.Get("Body"))
	twiml(w, "")
	if from == "" || body == "" {
		return
	}
	v := records.Visitor{Access: records.Public, Name: "Someone texting from " + from, Device: "phone", Login: "call:" + from + "|" + to}
	go s.answerText(v, from, to, body)
}

// answerText answers a text in its conversation, and sends the answer.
func (s *Server) answerText(v records.Visitor, from, to, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	key := "texts:said:" + from
	var said []chat.Said
	json.Unmarshal([]byte(s.app.Store.Meta(key)), &said)
	said = append(said, chat.Said{Text: trim.Clip(body, 1600)})
	answer, err := s.app.Chat.For(v).AnswerChat(ctx, "a text message to a phone: short, plain text, no Markdown, no lists", said)
	if err != nil || strings.TrimSpace(answer) == "" {
		answer = "Sorry, I could not answer just now."
	}
	said = append(said, chat.Said{Ours: true, Text: answer})
	if len(said) > textsKept {
		said = said[len(said)-textsKept:]
	}
	raw, _ := json.Marshal(said)
	s.app.Store.SetMeta(key, string(raw))
	if acct, ok := s.twilio(); ok {
		acct.Text(ctx, to, from, trim.Clip(plainText(answer), 1600)) // telegram.go: Markdown as plain words
	}
}
