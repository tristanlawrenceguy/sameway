package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// fakeTelegram is Telegram's Bot API for a test, reached in this process:
// updates queued for the next look, and what the bot sent.
type fakeTelegram struct {
	mu      sync.Mutex
	updates []map[string]any
	sent    []map[string]any
	next    int
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var in map[string]any
	json.NewDecoder(r.Body).Decode(&in)
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	out := map[string]any{"ok": true, "result": true}
	switch method {
	case "getMe":
		if !strings.Contains(r.URL.Path, "/bot123:abc/") {
			w.WriteHeader(http.StatusUnauthorized)
			out = map[string]any{"ok": false, "description": "Unauthorized"}
		} else {
			out["result"] = map[string]any{"id": 1, "username": "tris_bot"}
		}
	case "getUpdates":
		out["result"], f.updates = f.updates, nil
	default:
		in["method"] = method
		f.sent = append(f.sent, in)
	}
	json.NewEncoder(w).Encode(out)
}

func (f *fakeTelegram) say(chat int64, from, text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	f.updates = append(f.updates, map[string]any{"update_id": f.next, "message": map[string]any{"chat": map[string]any{"id": chat}, "from": map[string]any{"id": chat, "first_name": from}, "text": text}})
}

func (f *fakeTelegram) press(chat int64, data string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	f.updates = append(f.updates, map[string]any{"update_id": f.next, "callback_query": map[string]any{"id": "p" + data, "from": map[string]any{"id": chat}, "message": map[string]any{"chat": map[string]any{"id": chat}}, "data": data}})
}

// to is what the bot sent to a chat, joined, waiting a moment for it.
func (f *fakeTelegram) to(chat float64, want string) string {
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		f.mu.Lock()
		var all []string
		for _, m := range f.sent {
			if m["chat_id"] == chat {
				b, _ := json.Marshal(m)
				all = append(all, string(b))
			}
		}
		f.mu.Unlock()
		got := strings.Join(all, "\n")
		if strings.Contains(got, want) || time.Now().After(deadline) {
			return got
		}
	}
}

// Telegram is one more way in: the owner pastes their bot's token and
// links themselves with a code; anyone else who writes is answered as a
// visitor to the published pages, and what they ask for is a request that
// reaches the owner in Telegram with Make it and No, which only the
// owner can press; the answer goes back to them there; and what the owner
// writes goes to their own assistant.
func TestTelegramIsOneMoreWayIn(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	srv := h.(*server.Server)
	tg := &fakeTelegram{}
	a.Automate.HTTP = &http.Client{Transport: served{tg}}
	if page := get(t, h, "/telegram").Body.String(); !strings.Contains(page, "@BotFather") {
		t.Fatalf("the page says how to make a bot: %s", truncate(page))
	}
	if page := after(t, h, postForm(t, h, "/telegram/connect", url.Values{"token": {"wrong"}})).Body.String(); !strings.Contains(page, "did not take that token") {
		t.Errorf("a wrong token says so: %s", truncate(page))
	}
	postForm(t, h, "/telegram/connect", url.Values{"token": {"123:abc"}})
	page := get(t, h, "/telegram").Body.String()
	i := strings.Index(page, "/start ")
	if i < 0 {
		t.Fatalf("the owner is given a code to link themselves: %s", truncate(page))
	}
	code := page[i+len("/start ") : i+len("/start ")+8]

	tg.say(100, "Tris", "/start "+code)
	tg.say(200, "Pat", "Hi, could you come on Tuesday at 3pm? I'm Pat.")
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Visit from Pat", "starts": "2026-03-17T15:00:00Z"}}),
		{Text: "I have asked Tris for Tuesday at 3pm; you will hear back here."},
	}}
	if err := srv.KeepTelegram(context.Background(), tonight); err != nil {
		t.Fatal(err)
	}
	if got := tg.to(100, "Linked"); !strings.Contains(got, "Linked") {
		t.Errorf("the owner is linked: %s", got)
	}
	if got := tg.to(200, "asked Tris"); !strings.Contains(got, "asked Tris") {
		t.Errorf("someone else is answered there: %s", got)
	}
	if n, _ := a.Store.Count("event"); n != 0 {
		t.Fatal("nothing they ask for is made without the owner")
	}
	asks := a.Chat.Suggestions()
	if len(asks) != 1 {
		t.Fatalf("what they asked for is a request: %d", len(asks))
	}
	if got := tg.to(100, "yes:"+asks[0].ID); !strings.Contains(got, "Make it") {
		t.Fatalf("and reaches the owner in Telegram with its answers to press: %s", got)
	}

	tg.press(200, "yes:"+asks[0].ID)
	tg.press(100, "yes:"+asks[0].ID)
	srv.KeepTelegram(context.Background(), tonight)
	tg.mu.Lock()
	refused, _ := json.Marshal(tg.sent)
	tg.mu.Unlock()
	if !strings.Contains(string(refused), `"callback_query_id":"pyes:`+asks[0].ID+`","method":"answerCallbackQuery","text":"Only the owner answers"`) {
		t.Errorf("only the owner presses: %s", refused)
	}
	if n, _ := a.Store.Count("event"); n != 1 {
		t.Fatal("the owner's press makes it")
	}
	if got := tg.to(200, "said yes"); !strings.Contains(got, "said yes to your request") {
		t.Errorf("and the answer goes back to them there: %s", got)
	}

	a.Chat.Provider = &scripted{steps: []*llm.Response{{Text: "Pat is coming on Tuesday at 3pm."}}}
	tg.say(100, "Tris", "What is on Tuesday?")
	srv.KeepTelegram(context.Background(), tonight)
	if got := tg.to(100, "Pat is coming"); !strings.Contains(got, "Pat is coming on Tuesday") {
		t.Errorf("what the owner writes goes to their own assistant: %s", got)
	}
}
