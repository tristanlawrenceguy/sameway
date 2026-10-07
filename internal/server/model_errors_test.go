package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A turn the provider refuses is said in words with what to do, and the
// newest failure offers Send again with what was asked; a model gone from
// its server brings the connect card back.
func TestAFailedTurnSaysWhatToDoAndOffersSendAgain(t *testing.T) {
	gone := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			io.WriteString(w, `{"data":[{"id":"other"}]}`)
			return
		}
		if gone {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":{"message":"model 'qwen' not found"}}`)
		}
	}))
	defer srv.Close()
	was := llm.DefaultCandidates
	llm.DefaultCandidates = nil
	defer func() { llm.DefaultCandidates = was }()
	a, h := newApp(t)
	a.Workspace.Config.LLM = llm.Config{Provider: "openai", BaseURL: srv.URL + "/v1", Model: "qwen"}
	a.Chat.Provider, a.Chat.ProviderErr = &llm.OpenAI{BaseURL: srv.URL + "/v1", Model: "qwen"}, nil

	page := after(t, h, postForm(t, h, "/chat", url.Values{"message": {"plan my week"}, "from": {"/chat"}})).Body.String()
	if !strings.Contains(page, "The model qwen is not there any more") {
		t.Errorf("the failure is said in words with what to do: %s", truncate(page))
	}
	if !strings.Contains(page, `class="sw-again"`) || !strings.Contains(page, `name="message" value="plan my week"`) || !strings.Contains(page, ">Send again<") {
		t.Errorf("the newest failure offers Send again with what was asked: %s", truncate(page))
	}
	if !strings.Contains(page, "The model qwen is not on this computer any more.") {
		t.Errorf("a model gone brings the connect card back: %s", truncate(page))
	}
}
