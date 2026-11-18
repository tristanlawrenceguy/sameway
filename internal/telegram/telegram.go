// Package telegram talks to a Telegram bot the person made with
// @BotFather: what was sent to it since it last looked, and what it says
// back, with buttons under a message when there is something to press.
// It is Telegram's Bot API, over https, and nothing else.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// API is where Telegram's Bot API answers.
const API = "https://api.telegram.org"

// Bot is one bot, by its token.
type Bot struct {
	Token string
	HTTP  *http.Client // nil is one with a minute's limit
	Base  string       // "" is API; a test gives its own
}

// User is who sent something.
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// Name is a user as a person reads them: Pat Lee (@pat).
func (u User) Name() string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if u.Username != "" {
		name += " (@" + u.Username + ")"
	}
	return strings.TrimSpace(name)
}

// Message is something said in a chat.
type Message struct {
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"` // private, group, supergroup or channel
	} `json:"chat"`
	From User   `json:"from"`
	Text string `json:"text"`
}

// Press is a button under a message pressed.
type Press struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// Update is one thing that happened, in order.
type Update struct {
	ID      int64    `json:"update_id"`
	Message *Message `json:"message"`
	Press   *Press   `json:"callback_query"`
}

// Button is one button under a message: what it says and what pressing
// it sends back.
type Button struct {
	Text string `json:"text"`
	Data string `json:"callback_data"`
}

func (b Bot) call(ctx context.Context, method string, in, out any) error {
	base, client := b.Base, b.HTTP
	if base == "" {
		base = API
	}
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/bot"+b.Token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return errors.New("Telegram could not be reached")
	}
	defer res.Body.Close()
	var got struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&got) != nil {
		return fmt.Errorf("Telegram answered %s", res.Status)
	}
	if !got.OK {
		if res.StatusCode == http.StatusUnauthorized {
			return errors.New("Telegram did not take that token: copy it again from @BotFather")
		}
		return fmt.Errorf("Telegram said: %s", got.Description)
	}
	if out != nil {
		return json.Unmarshal(got.Result, out)
	}
	return nil
}

// Me is the bot's own username, which says the token works.
func (b Bot) Me(ctx context.Context) (string, error) {
	var me User
	err := b.call(ctx, "getMe", map[string]any{}, &me)
	return me.Username, err
}

// Updates are what happened after offset, waiting up to wait for one.
func (b Bot) Updates(ctx context.Context, offset int64, wait time.Duration) ([]Update, error) {
	var out []Update
	err := b.call(ctx, "getUpdates", map[string]any{"offset": offset, "timeout": int(wait.Seconds()), "allowed_updates": []string{"message", "callback_query"}}, &out)
	return out, err
}

// Send says text in a chat, with rows of buttons under it, or none.
func (b Bot) Send(ctx context.Context, chat int64, text string, buttons ...[]Button) error {
	in := map[string]any{"chat_id": chat, "text": text}
	if len(buttons) > 0 {
		in["reply_markup"] = map[string]any{"inline_keyboard": buttons}
	}
	return b.call(ctx, "sendMessage", in, nil)
}

// Answer stops a pressed button's spinner, with a few words.
func (b Bot) Answer(ctx context.Context, press, text string) error {
	return b.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": press, "text": text}, nil)
}
