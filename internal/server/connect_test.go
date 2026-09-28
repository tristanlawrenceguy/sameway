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

// A first run with no model the assistant can reach does not end in a
// connection error: the conversation says, in plain words, what is wrong
// and offers what is on this computer, each with where the conversation
// would go. One press connects it, and the card goes.
func TestAFirstRunOffersTheModelsOnThisComputer(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			io.WriteString(w, `{"data":[{"id":"llama3.1"},{"id":"qwen3"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer ollama.Close()
	was := llm.DefaultCandidates
	llm.DefaultCandidates = []llm.Candidate{{Server: "Ollama", BaseURL: ollama.URL + "/v1"}}
	defer func() { llm.DefaultCandidates = was }()

	a, h := newApp(t)
	page := get(t, h, "/chat").Body.String()
	for _, want := range []string{"Connect the assistant to an AI model", "No AI model is set up yet.", "Use Ollama (llama3.1)", "Your conversations and notes stay here.", "Check again"} {
		if !strings.Contains(page, want) {
			t.Errorf("the first run says %q; body: %s", want, truncate(page))
		}
	}
	if strings.Contains(page, "workspace.yaml") {
		t.Error("nobody is sent to edit a file")
	}

	r := postForm(t, h, "/model/use", url.Values{"choice": {"server " + ollama.URL + "/v1"}, "from": {"/chat"}})
	after := after(t, h, r).Body.String()
	if !strings.Contains(after, "Ollama (llama3.1) is the assistant") {
		t.Errorf("connecting says so; body: %s", truncate(after))
	}
	if strings.Contains(after, "Connect the assistant to an AI model") {
		t.Error("once connected, the card is gone")
	}
	yaml, _ := os.ReadFile(filepath.Join(a.Workspace.Dir, "workspace.yaml"))
	if !strings.Contains(string(yaml), "base_url: "+ollama.URL+"/v1") || !strings.Contains(string(yaml), "model: llama3.1") {
		t.Errorf("the choice is kept in the workspace:\n%s", yaml)
	}
	if a.Chat.Provider == nil || a.Chat.Provider.Name() == "" {
		t.Error("the next message goes to the model chosen")
	}
}

// When the model stops answering, the card says so in words a person
// reads, and when nothing is found it says what to install.
func TestAModelThatStoppedAnsweringIsSaidPlainly(t *testing.T) {
	was := llm.DefaultCandidates
	llm.DefaultCandidates = nil
	defer func() { llm.DefaultCandidates = was }()
	gone := httptest.NewServer(http.NotFoundHandler())
	base := gone.URL + "/v1"
	gone.Close()

	a, h := newApp(t)
	a.Workspace.Config.LLM = llm.Config{Provider: "openai", BaseURL: base, Model: "llama3.1"}
	a.Chat.Provider, a.Chat.ProviderErr = llm.New(a.Workspace.Config.LLM)
	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, "The AI model at "+base+" isn&#39;t answering. It may not be running.") {
		t.Errorf("the card says the model is not answering; body: %s", truncate(page))
	}
	// Claude Code, when this computer has it, is offered; otherwise the
	// card says what to install.
	if !strings.Contains(page, "ollama.com") && !strings.Contains(page, "Use Claude") {
		t.Errorf("with no model server running, the card offers a way; body: %s", truncate(page))
	}
}

// With no model yet, a person can still make their first note: one press
// makes it and opens its editor, its name ready to change.
func TestAFirstNoteCanBeMadeByHand(t *testing.T) {
	a, h := newApp(t)
	r := postForm(t, h, "/t/note/add", url.Values{})
	if r.Code != http.StatusSeeOther || !strings.HasSuffix(r.Header().Get("Location"), "#edit") {
		t.Fatalf("adding goes to the new note, open to edit, got %d %q", r.Code, r.Header().Get("Location"))
	}
	notes, _ := a.Store.Count("note")
	if notes != 1 {
		t.Errorf("one note is made, got %d", notes)
	}
	page := get(t, h, strings.TrimSuffix(r.Header().Get("Location"), "#edit")).Body.String()
	if !strings.Contains(page, ">New note</h1>") || !strings.Contains(page, "template data-edit-fields") {
		t.Errorf("the new note has its name to change and everything to edit; body: %s", truncate(page))
	}
}
