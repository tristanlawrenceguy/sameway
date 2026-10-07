package chat_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A model on this computer is readied before the person's first message:
// the instructions, the tools and the conversation are sent ahead for one
// word, and Ollama is asked to keep the model; not again within minutes,
// and never for a model elsewhere.
func TestAModelHereIsReadiedAhead(t *testing.T) {
	var asked []string
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var b map[string]any
		json.Unmarshal(body, &b)
		switch r.URL.Path {
		case "/v1/chat/completions":
			msgs, _ := b["messages"].([]any)
			first, _ := msgs[0].(map[string]any)
			asked = append(asked, "completion max_tokens="+jsonNum(b["max_tokens"])+" system="+map[bool]string{true: "yes", false: "no"}[strings.Contains(first["content"].(string), "Sameway workspace")])
			io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"."},"finish_reason":"length"}]}`)
		case "/api/generate":
			asked = append(asked, "keep "+b["keep_alive"].(string))
		}
	}))
	defer ollama.Close()
	was := llm.OllamaURL
	llm.OllamaURL = ollama.URL
	defer func() { llm.OllamaURL = was }()

	svc := newFullService(t)
	svc.Provider = &llm.OpenAI{BaseURL: ollama.URL + "/v1", Model: "warm-test-model"}
	if !svc.Warm(context.Background()) {
		t.Fatal("a model here is readied")
	}
	if strings.Join(asked, "|") != "completion max_tokens=1 system=yes|keep 30m" {
		t.Errorf("the instructions are sent ahead for one word and the model kept: %v", asked)
	}
	if svc.Warm(context.Background()) {
		t.Error("not again within minutes")
	}
	svc.Provider = &llm.OpenAI{BaseURL: "https://openrouter.ai/api/v1", Model: "elsewhere"}
	if svc.Warm(context.Background()) {
		t.Error("a model elsewhere is not readied")
	}
}

func jsonNum(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
