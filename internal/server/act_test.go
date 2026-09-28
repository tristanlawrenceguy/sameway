package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person's action is a button: on the action's own page, on the canvas
// where the assistant put it, and for an agent over the API. Pressing it
// runs the webhook, lands the person back where they were, and the log
// says so.
func TestAnActionIsAButtonEverywhere(t *testing.T) {
	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		io.WriteString(w, "alarm on")
	}))
	defer remote.Close()
	chat.HTTPClient = remote.Client()

	a, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/action", map[string]any{"title": "Turn on the alarm", "url": remote.URL + "/alarm"})
	wantStatus(t, created, http.StatusCreated)
	var action struct{ ID string }
	decode(t, created, &action)

	// Its own page has a Run button that returns there.
	page := get(t, h, "/t/action/"+action.ID).Body.String()
	if !strings.Contains(page, `action="/act/`+action.ID+`"`) || !strings.Contains(page, "Run") {
		t.Fatal("an action's page should carry its Run button")
	}
	rec := postForm(t, h, "/act/"+action.ID, url.Values{"from": {"/t/action/" + action.ID}})
	wantStatus(t, rec, http.StatusSeeOther)
	if loc := rec.Header().Get("Location"); loc != "/t/action/"+action.ID || calls != 1 {
		t.Errorf("pressing Run should call the webhook once and return to the page, got %q after %d calls", loc, calls)
	}
	if !logged(t, h, "You ran action Turn on the alarm (200)") {
		t.Error("the run should be in the log under the person's name")
	}

	// The assistant puts it on the canvas as a button block; the page
	// renders it as a form a person can press.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "button", "props": map[string]any{"label": "Turn on the alarm", "action": action.ID}}),
		{Text: "There is your button."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"give me a button for the alarm"}, "from": {"/"}})
	canvas := get(t, h, "/").Body.String()
	if !strings.Contains(canvas, `<form method="post" action="/act/`+action.ID+`"`) {
		t.Error("the button block should be a form that runs the action")
	}

	// An agent runs it over the API and reads the answer.
	rec = postJSON(t, h, http.MethodPost, "/api/act/"+action.ID, nil)
	wantStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "alarm on") || calls != 2 {
		t.Errorf("the API should run it and answer with the result, got %s after %d calls", rec.Body.String(), calls)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/act/nope", nil), http.StatusBadRequest)
}
