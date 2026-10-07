package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// A saved key the provider has taken back makes the model not answer,
// said so a new one can be pasted; asked again only after a while.
func TestASavedKeyTakenBackIsSaid(t *testing.T) {
	t.Setenv("SAMEWAY_KEYS", filepath.Join(t.TempDir(), "keys.json"))
	t.Setenv("ANTHROPIC_API_KEY", "")
	asked := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer provider.Close()
	was := AnthropicKeyURL
	AnthropicKeyURL = provider.URL
	defer func() { AnthropicKeyURL = was }()
	if err := SaveKey("ANTHROPIC_API_KEY", "sk-ant-taken-back"); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Provider: "anthropic", APIKeyEnv: "ANTHROPIC_API_KEY"}
	ok, why := Answers(context.Background(), cfg)
	if ok || !strings.Contains(why, "Anthropic no longer accepts the key") {
		t.Errorf("a key taken back is said: %v %q", ok, why)
	}
	Answers(context.Background(), cfg)
	if asked != 1 {
		t.Errorf("asked once in a while, not on every page: %d", asked)
	}
}
