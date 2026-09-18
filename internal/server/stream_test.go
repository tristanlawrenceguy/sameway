package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// With scripts the composer posts to /chat/stream and hears the turn as it
// happens: the person's message as recorded, each tool as it starts, each
// block as it lands with its HTML, and the reply as recorded, as
// server-sent events. The page can then show all of it without waiting.
func TestATurnIsToldAsItHappens(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Added a card called Shopping."},
	}}, nil

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("message", "add a shopping card")
	mw.WriteField("from", "/")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("the turn comes as server-sent events, got %q", ct)
	}
	out := rec.Body.String()
	// The stream is JSON, so quotes inside the HTML come escaped.
	order := []string{"event: said", "event: tool", `"label":"Adding a card"`, "event: change", `data-block-id=`, `data-changed=\"added\"`, "event: text", "Added a card called Shopping.", "event: done", `sw-message--assistant`}
	at := -1
	for _, want := range order {
		i := strings.Index(out, want)
		if i < 0 {
			t.Errorf("the stream should carry %s\n%s", want, out)
			continue
		}
		if i < at {
			t.Errorf("%s comes out of order", want)
		}
		at = i
	}
	if !strings.Contains(out, `sw-status--done`) && !strings.Contains(out, `"status":"`) {
		t.Error("the end of the turn carries the status line for the page")
	}
	// The turn is recorded exactly as a plain post would record it.
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, "Added a card called Shopping.") {
		t.Error("the reply is on the page afterwards")
	}
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `data-block-component="card"`) {
		t.Error("the block is on the canvas afterwards")
	}
}
