package server_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// gated is a model that answers when it is let go.
type gated struct{ started, release chan struct{} }

func (g *gated) Name() string { return "gated" }
func (g *gated) Complete(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	close(g.started)
	<-g.release
	return &llm.Response{Text: "Here it is, done while you were away."}, nil
}

// Going to another page mid-turn does not stop the assistant. The turn
// runs on, the page gone to says the assistant is working and carries
// the turn's id, and /chat/live tells that page the turn from its start
// to its reply.
func TestATurnGoesOnWhenThePersonGoesElsewhere(t *testing.T) {
	a, h := newApp(t)
	model := &gated{started: make(chan struct{}), release: make(chan struct{})}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("message", "make me something")
	mw.WriteField("from", "/")
	mw.Close()
	ctx, leave := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", &body).WithContext(ctx)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	gone := make(chan struct{})
	go func() { h.ServeHTTP(httptest.NewRecorder(), req); close(gone) }()

	<-model.started
	leave()
	<-gone

	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `data-turn="turn-`) {
		t.Error("a page made mid-turn carries the turn's id, so it can follow it")
	}
	if !strings.Contains(page, "Assistant is working") || !strings.Contains(page, "make me something") {
		t.Error("a page made mid-turn shows the message and says the assistant is working")
	}

	live := httptest.NewRecorder()
	followed := make(chan struct{})
	go func() {
		h.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/chat/live?from=/chat", nil))
		close(followed)
	}()
	close(model.release)
	<-followed

	out := live.Body.String()
	if !strings.Contains(out, "event: said") || !strings.Contains(out, "event: done") || !strings.Contains(out, "done while you were away") {
		t.Errorf("the page gone to hears the turn from its start to its reply, got\n%s", out)
	}
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, "done while you were away") || strings.Contains(page, "data-turn=") {
		t.Error("afterwards the reply is on the page and no turn is under way")
	}
	if rec := get(t, h, "/chat/live"); rec.Code != http.StatusNoContent {
		t.Errorf("with no turn under way there is nothing to follow, got %d", rec.Code)
	}
}

// ask posts a message to /chat/stream as a page would, under ctx, and
// says when the stream is over.
func ask(h http.Handler, ctx context.Context, message string) <-chan struct{} {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("message", message)
	mw.WriteField("from", "/")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", &body).WithContext(ctx)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	over := make(chan struct{})
	go func() { h.ServeHTTP(httptest.NewRecorder(), req); close(over) }()
	return over
}

// A turn that ends with no page to hear it is told beyond the page, the
// way a ringing reminder is: the reply's words and the way back to them.
func TestATurnNobodyWatchedSaysItIsDone(t *testing.T) {
	a, h := newApp(t)
	told := make(chan string, 1)
	h.(*server.Server).OnRing(func(title, text, url string) { told <- title + " | " + text + " | " + url })
	model := &gated{started: make(chan struct{}), release: make(chan struct{})}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	ctx, leave := context.WithCancel(context.Background())
	gone := ask(h, ctx, "make me something")
	<-model.started
	leave()
	<-gone
	close(model.release)

	got := <-told
	if !strings.HasPrefix(got, "Assistant replied | Here it is, done while you were away. | ") || !strings.Contains(got, "/#msg-") {
		t.Errorf("the person hears the turn is done, with the reply and the way back to it, got %q", got)
	}
}

// A page that follows the turn to its end tells the person itself, so
// the server says nothing: the news comes once.
func TestATurnAPageWatchedIsNotToldTwice(t *testing.T) {
	a, h := newApp(t)
	told := make(chan string, 1)
	h.(*server.Server).OnRing(func(title, text, url string) { told <- title })
	model := &gated{started: make(chan struct{}), release: make(chan struct{})}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	over := ask(h, context.Background(), "make me something")
	<-model.started
	close(model.release)
	<-over

	select {
	case got := <-told:
		t.Errorf("a page heard the turn end, so nothing more is sent, got %q", got)
	case <-time.After(200 * time.Millisecond):
	}
}
