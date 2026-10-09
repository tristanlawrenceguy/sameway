package server_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// changesMade is the newest reply's Changes made list on a page.
func changesMade(page string) string {
	i := strings.LastIndex(page, `<ul class="sw-message__changes">`)
	if i < 0 {
		return ""
	}
	return page[i : i+strings.Index(page[i:], "</ul>")]
}

// eventSentence matches what an event says, its sentence, without its time
// or its Undo: a line under a reply and an entry in the log alike.
var eventSentence = regexp.MustCompile(`class="sw-event__text[^"]*">(.*?)</(?:span|h[234])>(?: <time|<form|</div>)`)

// sentences is what each event in some HTML says, in order.
func sentences(html string) []string {
	var out []string
	for _, m := range eventSentence.FindAllStringSubmatch(html, -1) {
		out = append(out, said(m[1]))
	}
	return out
}

// A change reads the same under the reply that made it and in the log,
// because both are the one line: a setting in words, a record by its
// kind and title with no suffix, and an undo quoting what it undid.
func TestChangesMadeSaysWhatTheLogSays(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.text", "value": "large"}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Seeds to buy"}}),
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Plan"}}),
		{Text: "Done."},
		toolCall("undo_change", map[string]any{}),
		{Text: "Taken back."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"bigger text, a note and a card"}, "from": {"/chat"}})

	chat := changesMade(get(t, h, "/chat").Body.String())
	got := sentences(chat)
	want := []string{"changed text size to Large", "created note Seeds to buy", "added card Plan"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("Changes made should say %q, got %q\n%s", want, got, chat)
	}
	if !strings.Contains(chat, `href="/t/note/`) || !strings.Contains(chat, `href="/canvas/`) {
		t.Errorf("each change should lead to what it made\n%s", chat)
	}
	if strings.Contains(chat, "ui.text") || strings.Contains(chat, "(note)") {
		t.Errorf("no keys and no type suffixes under the reply\n%s", chat)
	}
	if strings.Count(chat, `class="sw-event__undo"`) != 3 || !strings.Contains(chat, `name="from" value="/chat"`) {
		t.Errorf("each change under the newest reply should offer Undo back to the chat\n%s", chat)
	}
	log := strings.Join(sentences(get(t, h, "/activity").Body.String()), "\n")
	for _, line := range want {
		if !strings.Contains(log, "Assistant "+line) {
			t.Errorf("the log should say %q, as the reply does\n%s", "Assistant "+line, log)
		}
	}

	// The assistant takes the card back: its reply and the log say so
	// alike, and the old reply loses its Undo.
	postForm(t, h, "/chat", url.Values{"message": {"take the card back"}, "from": {"/chat"}})
	page := get(t, h, "/chat").Body.String()
	undone := sentences(changesMade(page))
	if len(undone) != 1 || undone[0] != "undid: Assistant added card Plan" {
		t.Fatalf("the undo should read as the log's, got %q", undone)
	}
	log = strings.Join(sentences(get(t, h, "/activity").Body.String()), "\n")
	if !strings.Contains(log, "Assistant undid: Assistant added card Plan") || strings.Contains(log, "undid::") {
		t.Errorf("the log should say the undo once, with one colon\n%s", log)
	}
	first := page[:strings.LastIndex(page, `<ul class="sw-message__changes">`)]
	if strings.Contains(changesMade(first), `class="sw-event__undo"`) {
		t.Error("an older reply keeps its links and loses its Undo")
	}
}

// A reply that arrives live is the same HTML as the reply on the page
// when it is loaded: one function makes both.
func TestAStreamedReplyIsTheRenderedReply(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "calm"}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Seeds to buy"}}),
		{Text: "Done."},
	}}, nil
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("message", "calm, and a note")
	mw.WriteField("from", "/chat")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var done struct{ ID, HTML string }
	for _, frame := range strings.Split(rec.Body.String(), "\n\n") {
		if data, ok := strings.CutPrefix(frame, "event: done\ndata: "); ok {
			if err := json.Unmarshal([]byte(data), &done); err != nil {
				t.Fatal(err)
			}
		}
	}
	if done.HTML == "" || !strings.Contains(done.HTML, "changed pace to Calm") && !strings.Contains(said(done.HTML), "changed pace to Calm") {
		t.Fatalf("the stream should end with the reply and its changes\n%s", rec.Body.String())
	}
	page := get(t, h, "/chat").Body.String()
	start := strings.Index(page, `<article class="sw-message sw-message--assistant"`)
	if start < 0 {
		t.Fatal("the reply should be on the page")
	}
	rendered := page[start : start+strings.Index(page[start:], "</article>")+len("</article>")]
	if rendered != done.HTML {
		t.Errorf("streamed and rendered replies differ\nstreamed: %s\nrendered: %s", done.HTML, rendered)
	}
}
