package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person with no model on this computer pastes a key made on a website:
// it is kept in their own settings, never in the workspace, the model is
// set to it, and a key Sameway does not know is refused with what to
// paste instead.
func TestAPersonPastesAKey(t *testing.T) {
	keys := filepath.Join(t.TempDir(), "keys.json")
	t.Setenv("SAMEWAY_KEYS", keys)
	was := llm.DefaultCandidates
	llm.DefaultCandidates = nil
	defer func() { llm.DefaultCandidates = was }()
	openrouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"limit_remaining":null}}`)
	}))
	defer openrouter.Close()
	wasURL := llm.OpenRouterKeyURL
	llm.OpenRouterKeyURL = openrouter.URL
	defer func() { llm.OpenRouterKeyURL = wasURL }()
	a, h := newApp(t)

	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `name="key"`) || !strings.Contains(page, `type="password"`) || !strings.Contains(page, "Or paste a key") {
		t.Errorf("the connect card offers to paste a key: %s", truncate(page))
	}

	bad := after(t, h, postForm(t, h, "/model/key", url.Values{"key": {"hello"}, "from": {"/chat"}})).Body.String()
	if !strings.Contains(bad, "begins sk-ant-") {
		t.Errorf("a key Sameway does not know says what to paste: %s", truncate(bad))
	}

	done := after(t, h, postForm(t, h, "/model/key", url.Values{"key": {"sk-or-v1-abc123"}, "from": {"/chat"}})).Body.String()
	if !strings.Contains(done, "OpenRouter is the assistant") {
		t.Errorf("connecting says so: %s", truncate(done))
	}
	yaml, _ := os.ReadFile(filepath.Join(a.Workspace.Dir, "workspace.yaml"))
	if strings.Contains(string(yaml), "sk-or-v1-abc123") || !strings.Contains(string(yaml), "openrouter.ai") {
		t.Errorf("the workspace names the service, never the key:\n%s", yaml)
	}
	if kept, _ := os.ReadFile(keys); !strings.Contains(string(kept), "sk-or-v1-abc123") || llm.Key("OPENROUTER_API_KEY") != "sk-or-v1-abc123" {
		t.Error("the key is kept in the person's own settings and read from there")
	}
}
