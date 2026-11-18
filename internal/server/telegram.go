package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/telegram"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Telegram, as one more way in: the owner makes a bot with @BotFather and
// pastes its token here, once. The owner links themselves by sending the
// bot a code, and then talks to their own assistant from their phone,
// and a request waiting for their yes comes with Make it and No to press.
// Anyone else who writes to the bot is answered as a visitor to the
// published pages is (chat/channel.go): their reach, what they ask to be
// made a request, the owner's answer told back to them there.

const (
	telegramKey  = "telegram:bot"   // the token, in the keys file
	telegramName = "telegram:name"  // the bot's username
	telegramCode = "telegram:code"  // what the owner sends to link themselves
	telegramMe   = "telegram:owner" // the owner's chat, once linked
	telegramAt   = "telegram:next"  // the next update to read
	telegramWait = 20 * time.Second // how long one look waits for news
	telegramKept = 12               // messages of a conversation kept
)

func (s *Server) bot() (telegram.Bot, bool) {
	token := s.keys().Get(telegramKey)
	return telegram.Bot{Token: token, HTTP: s.app.Automate.HTTP}, token != ""
}

func (s *Server) telegramPage(w http.ResponseWriter, r *http.Request) {
	esc := template.HTMLEscapeString
	var b strings.Builder
	if _, ok := s.bot(); ok {
		name := s.app.Store.Meta(telegramName)
		if s.app.Store.Meta(telegramMe) == "" {
			b.WriteString(`<p>Connected to <a class="sw-link" href="https://t.me/` + esc(name) + `">@` + esc(name) + `</a>. To link yourself, send it this message:</p><p><code>/start ` + esc(s.app.Store.Meta(telegramCode)) + `</code></p>`)
		} else {
			b.WriteString(`<p>Connected to <a class="sw-link" href="https://t.me/` + esc(name) + `">@` + esc(name) + `</a>, and linked to you. What you send it goes to your assistant, and requests come there with Make it and No. Anyone else who writes to it is answered as a visitor to your published pages, and what they ask for waits for your yes.</p>`)
		}
		b.WriteString(string(s.form(ui.Form{Action: "/telegram/off", Button: &ui.Button{Label: "Stop using Telegram", Variant: ui.Secondary}})))
		s.page(w, r, "Telegram", template.HTML(b.String()), pageOptions{})
		return
	}
	b.WriteString(`<ol class="sw-stack"><li>In Telegram, open <a class="sw-link" href="https://t.me/BotFather">@BotFather</a> and send <code>/newbot</code>.</li><li>Give it a name, and a username ending in bot.</li><li>Paste the token it gives you here.</li></ol>`)
	b.WriteString(string(s.form(ui.Form{Action: "/telegram/connect", Class: "sw-stack",
		Body:   s.part(ui.TextField{Label: "The bot's token", Name: "token", Required: true, Hint: "It looks like 123456:ABC-DEF. Keep it to yourself: whoever has it speaks as your bot."}),
		Button: &ui.Button{Label: "Connect"}})))
	s.page(w, r, "Telegram", template.HTML(b.String()), pageOptions{Lede: "Your assistant in Telegram, and a way for anyone to ask you."})
}

func (s *Server) telegramConnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	bot := telegram.Bot{Token: strings.TrimSpace(r.PostForm.Get("token")), HTTP: s.app.Automate.HTTP}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	name, err := bot.Me(ctx)
	if err == nil {
		err = s.keys().Save(telegramKey, bot.Token)
	}
	if err != nil {
		s.failed(w, r, "Not connected", err, "/telegram")
		return
	}
	code := make([]byte, 4)
	rand.Read(code)
	s.app.Store.SetMeta(telegramName, name)
	s.app.Store.SetMeta(telegramCode, hex.EncodeToString(code))
	s.app.Store.SetMeta(telegramMe, "")
	s.app.Store.SetMeta(telegramAt, "") // another bot counts its updates afresh
	s.tellAt(w, r, outcome{Title: "Telegram connected", Text: "Send @" + name + " the message on this page to link yourself."}, "/telegram")
}

func (s *Server) telegramOff(w http.ResponseWriter, r *http.Request) {
	s.keys().Save(telegramKey, "")
	s.app.Store.SetMeta(telegramMe, "")
	s.tellAt(w, r, outcome{Title: "Telegram no longer used", Text: "The bot no longer answers."}, "/telegram")
}

// KeepTelegram reads what was sent to the bot and answers it: a
// background job, which waits a while for news each time it looks.
func (s *Server) KeepTelegram(ctx context.Context, _ time.Time) error {
	bot, ok := s.bot()
	if !ok {
		return nil
	}
	next, _ := strconv.ParseInt(s.app.Store.Meta(telegramAt), 10, 64)
	updates, err := bot.Updates(ctx, next, telegramWait)
	if err != nil {
		return err
	}
	for _, u := range updates {
		s.app.Store.SetMeta(telegramAt, strconv.FormatInt(u.ID+1, 10))
		switch {
		case u.Press != nil:
			s.telegramPressed(ctx, bot, u.Press)
		case u.Message != nil && strings.TrimSpace(u.Message.Text) != "":
			s.telegramSaid(ctx, bot, u.Message)
		}
	}
	return nil
}

// telegramSaid answers one message: the owner's goes to their assistant,
// anyone else's is answered with a visitor's reach.
func (s *Server) telegramSaid(ctx context.Context, bot telegram.Bot, m *telegram.Message) {
	chatID, text := m.Chat.ID, strings.TrimSpace(m.Text)
	me := s.app.Store.Meta(telegramMe)
	// Only from the owner's own chat with the bot: linked in a group,
	// everyone in it would speak as the owner.
	if code := s.app.Store.Meta(telegramCode); me == "" && code != "" && text == "/start "+code && m.Chat.Type == "private" {
		s.app.Store.SetMeta(telegramMe, strconv.FormatInt(chatID, 10))
		bot.Send(ctx, chatID, "Linked. What you send here goes to your assistant, and requests come here for your yes.")
		return
	}
	if me == strconv.FormatInt(chatID, 10) {
		rec, err := s.app.Chat.Send(ctx, text)
		answer := ""
		if err != nil {
			answer = err.Error()
		} else {
			answer, _ = rec.Fields["content"].(string)
		}
		bot.Send(ctx, chatID, trim.Clip(answer, 4000))
		return
	}
	v := records.Visitor{Access: records.Public, Name: m.From.Name(), Device: "telegram", Login: "tg:" + strconv.FormatInt(chatID, 10)}
	key := "telegram:said:" + strconv.FormatInt(chatID, 10)
	var said []chat.Said
	json.Unmarshal([]byte(s.app.Store.Meta(key)), &said)
	said = append(said, chat.Said{Text: trim.Clip(text, 4000)})
	answer, err := s.app.Chat.For(v).AnswerChat(ctx, "Telegram", said)
	if err != nil {
		answer = "Sorry, I could not answer just now."
	}
	said = append(said, chat.Said{Ours: true, Text: answer})
	if len(said) > telegramKept {
		said = said[len(said)-telegramKept:]
	}
	raw, _ := json.Marshal(said)
	s.app.Store.SetMeta(key, string(raw))
	bot.Send(ctx, chatID, trim.Clip(answer, 4000))
}

// telegramPressed answers a request from the owner's chat: Make it or No.
func (s *Server) telegramPressed(ctx context.Context, bot telegram.Bot, p *telegram.Press) {
	if p.Message == nil || strconv.FormatInt(p.Message.Chat.ID, 10) != s.app.Store.Meta(telegramMe) {
		bot.Answer(ctx, p.ID, "Only the owner answers")
		return
	}
	verb, id, _ := strings.Cut(p.Data, ":")
	var err error
	said := "Made"
	switch verb {
	case "yes":
		err = s.app.Chat.Accept(id)
	case "no":
		err, said = s.app.Chat.Dismiss(id), "Not made"
	default:
		err = errors.New("that button does nothing")
	}
	if err != nil {
		said = err.Error()
	}
	s.Changed()
	bot.Answer(ctx, p.ID, said)
}

// telegramAsk puts a request that waits to the owner in Telegram, with
// its answers to press, when they are linked.
func (s *Server) telegramAsk(summary string) {
	bot, ok := s.bot()
	me, err := strconv.ParseInt(s.app.Store.Meta(telegramMe), 10, 64)
	if !ok || err != nil {
		return
	}
	for _, p := range s.app.Chat.Suggestions() {
		if p.Fields["summary"] == summary {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			bot.Send(ctx, me, summary, []telegram.Button{{Text: "Make it", Data: "yes:" + p.ID}, {Text: "No", Data: "no:" + p.ID}})
			return
		}
	}
}

// telegramTell tells someone who asked in Telegram what the owner answered.
func (s *Server) telegramTell(v records.Visitor, text string) bool {
	chatID, err := strconv.ParseInt(strings.TrimPrefix(v.Login, "tg:"), 10, 64)
	bot, ok := s.bot()
	if v.Device != "telegram" || err != nil || !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return bot.Send(ctx, chatID, text) == nil
}
