package server_test

import (
	"bytes"
	"context"
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

// streamed is a scripted model that streams: it names each tool it is
// about to call before the call is whole, the way a real stream does.
type streamed struct{ scripted }

func (s *streamed) Stream(ctx context.Context, req llm.Request, on func(llm.Delta)) (*llm.Response, error) {
	resp, err := s.Complete(ctx, req)
	if err == nil {
		for _, c := range resp.ToolCalls {
			on(llm.Delta{Call: c.Name})
		}
		if resp.Text != "" {
			on(llm.Delta{Text: resp.Text})
		}
	}
	return resp, err
}

// A model that streams names a tool as soon as it starts to call it, so
// the page can show the step while the arguments are still being written;
// the same step is told again, in full, when the call runs.
func TestAToolIsNamedBeforeItsCallIsWhole(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &streamed{scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Added a card called Shopping."},
	}}}, nil

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("message", "add a shopping card")
	mw.WriteField("from", "/")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	out := rec.Body.String()
	early := strings.Index(out, `"early":true`)
	full := strings.Index(out, `"label":"Adding a card"`)
	if early < 0 || full < 0 || early > full {
		t.Errorf("an early step names the tool before the full one, got\n%s", out)
	}
	if !strings.Contains(out[:full], `"label":"Adding a block"`) || !strings.Contains(out[:full], `"tool":"add_component"`) {
		t.Errorf("the early step carries the tool's name and a broad label\n%s", out)
	}
	if strings.Count(out, `"early":true`) != 1 {
		t.Errorf("the reply's words are not an early step\n%s", out)
	}
}
