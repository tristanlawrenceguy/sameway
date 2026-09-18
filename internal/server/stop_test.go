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

// waiting is a model that answers only when its turn is stopped.
type waiting struct{ started chan struct{} }

func (w *waiting) Name() string { return "waiting" }
func (w *waiting) Complete(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	close(w.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

// A person can stop a turn from the page. The stream ends with a reply
// that says so, recorded like any other, and the turn is not an error.
func TestAPersonCanStopATurn(t *testing.T) {
	a, h := newApp(t)
	model := &waiting{started: make(chan struct{})}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("message", "take your time")
	mw.WriteField("from", "/")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	finished := make(chan struct{})
	go func() { h.ServeHTTP(rec, req); close(finished) }()

	<-model.started
	stop := httptest.NewRequest(http.MethodPost, "/chat/stop", strings.NewReader("turn="))
	stop.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stopRec := httptest.NewRecorder()
	h.ServeHTTP(stopRec, stop)
	if stopRec.Code != http.StatusNoContent {
		t.Errorf("stop answers 204, got %d", stopRec.Code)
	}
	<-finished

	out := rec.Body.String()
	if !strings.Contains(out, "event: done") || !strings.Contains(out, "Stopped, as you asked.") || strings.Contains(out, "event: error") {
		t.Errorf("the stream ends with a reply that says the turn was stopped, got\n%s", out)
	}
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, "Stopped, as you asked.") {
		t.Error("the stopped reply is on the page afterwards")
	}
}
