package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A pasted key is asked about before it is kept: one the provider does not
// know, or one with no credit, is said to be wrong now, kept nowhere; one
// that could not be asked about is kept, saying so.
func TestAPastedKeyIsCheckedFirst(t *testing.T) {
	t.Setenv("SAMEWAY_KEYS", filepath.Join(t.TempDir(), "keys.json"))
	was := llm.DefaultCandidates
	llm.DefaultCandidates = nil
	defer func() { llm.DefaultCandidates = was }()
	answer := http.StatusUnauthorized
	body := `{}`
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(answer)
		io.WriteString(w, body)
	}))
	defer provider.Close()
	wasA, wasO := llm.AnthropicKeyURL, llm.OpenRouterKeyURL
	llm.AnthropicKeyURL, llm.OpenRouterKeyURL = provider.URL, provider.URL
	defer func() { llm.AnthropicKeyURL, llm.OpenRouterKeyURL = wasA, wasO }()
	_, h := newApp(t)
	paste := func(key string) string {
		return after(t, h, postForm(t, h, "/model/key", url.Values{"key": {key}, "from": {"/chat"}})).Body.String()
	}

	if page := paste("sk-ant-wrong"); !strings.Contains(page, "Anthropic did not accept this key") || llm.Key("ANTHROPIC_API_KEY") != "" {
		t.Errorf("a refused key is said and not kept: %s", truncate(page))
	}
	answer, body = http.StatusOK, `{"data":{"limit_remaining":0}}`
	if page := paste("sk-or-empty"); !strings.Contains(page, "no credit left") || llm.Key("OPENROUTER_API_KEY") != "" {
		t.Errorf("a key with no credit is said and not kept: %s", truncate(page))
	}
	answer = http.StatusServiceUnavailable
	if page := paste("sk-ant-maybe"); !strings.Contains(page, "could not be checked") || llm.Key("ANTHROPIC_API_KEY") != "sk-ant-maybe" {
		t.Errorf("a key that could not be checked is kept, saying so: %s", truncate(page))
	}
}
