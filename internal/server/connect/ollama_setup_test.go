package connect_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A person with Ollama and no model presses one button: Sameway fetches the
// free model through Ollama, gives it room for its prompt, and makes it the
// assistant's model. No terminal, nothing unsigned.
func TestAPersonFetchesAFreeModelThroughOllama(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	have, created := false, ""
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/api/version":
			io.WriteString(w, `{"version":"0.35.1"}`)
		case "/v1/models":
			if have {
				io.WriteString(w, `{"data":[{"id":"qwen3.5:4b"}]}`)
			} else {
				io.WriteString(w, `{"data":[]}`)
			}
		case "/api/pull":
			io.WriteString(w, `{"status":"pulling","total":100,"completed":40}`+"\n"+`{"status":"success"}`+"\n")
			have = true
		case "/api/create":
			b, _ := io.ReadAll(r.Body)
			created = string(b)
			io.WriteString(w, `{"status":"success"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()
	a, h := newAppWith(t, app.Options{Ollama: llm.Ollama{URL: ollama.URL}, LocalModels: []llm.Candidate{{Server: "Ollama", BaseURL: ollama.URL + "/v1"}}})
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, "Fetch a free model ("+llm.FreeModelWords+")") {
		i := strings.Index(page, "sw-connect")
		t.Fatalf("Ollama with no model offers the free one: %s", page[max(i, 0):min(max(i, 0)+3000, len(page))])
	}
	postForm(t, h, "/model/ollama", url.Values{"from": {"/chat"}})
	yaml := filepath.Join(a.Workspace.Dir, "workspace.yaml")
	deadline := time.Now().Add(5 * time.Second)
	for {
		b, _ := os.ReadFile(yaml)
		if strings.Contains(string(b), "model: sameway-qwen3.5") || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	b, _ := os.ReadFile(yaml)
	if !strings.Contains(string(b), "model: sameway-qwen3.5") || !strings.Contains(string(b), "base_url: "+ollama.URL+"/v1") {
		t.Errorf("once fetched, Sameway's copy is the assistant's model:\n%s", b)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(created, `"num_ctx":32768`) || !strings.Contains(created, `"from":"qwen3.5:4b"`) {
		t.Errorf("the copy has room for Sameway's prompt: %s", created)
	}
}

// A fetch is the workspace's: a second server on it says how far it has
// come, and another workspace with Ollama of its own is still offered one.
func TestAFetchIsEachWorkspacesOwn(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	ollama := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/version":
				io.WriteString(w, `{"version":"0.35.1"}`)
			case "/v1/models":
				io.WriteString(w, `{"data":[]}`)
			case "/api/pull":
				io.WriteString(w, `{"status":"pulling","total":100,"completed":40}`+"\n")
				w.(http.Flusher).Flush()
				<-release
				io.WriteString(w, `{"error":"stopped"}`+"\n")
			default:
				http.NotFound(w, r)
			}
		}))
	}
	mine, theirs := ollama(), ollama()
	defer mine.Close()
	defer theirs.Close()
	defer close(release)
	optsAt := func(o *httptest.Server) app.Options {
		return app.Options{Ollama: llm.Ollama{URL: o.URL}, LocalModels: []llm.Candidate{{Server: "Ollama", BaseURL: o.URL + "/v1"}}}
	}
	a, h := newAppWith(t, optsAt(mine))
	_, other := newAppWith(t, optsAt(theirs))
	postForm(t, h, "/model/ollama", url.Values{"from": {"/chat"}})
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(get(t, server.New(a), "/chat").Body.String(), "Fetching the free model: 40%") {
		if time.Now().After(deadline) {
			t.Fatalf("a second server on the workspace says how far it has come: %s", truncate(get(t, server.New(a), "/chat").Body.String()))
		}
		time.Sleep(50 * time.Millisecond)
	}
	if page := get(t, other, "/chat").Body.String(); strings.Contains(page, "Fetching the free model:") || !strings.Contains(page, "Fetch a free model") {
		t.Errorf("another workspace is offered its own: %s", truncate(page))
	}
}
