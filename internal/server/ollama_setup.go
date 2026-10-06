package server

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person with Ollama and no model was told to open a terminal and type
// ollama pull. Now the connect card offers the model with one press: Sameway
// fetches it through Ollama's own API, the card says how far it has come,
// and when it is here it is given room for Sameway's prompt and becomes the
// assistant's model. Nothing leaves the computer, and nothing unsigned runs.

// fetchState is how a fetch is coming along.
type fetchState struct {
	sync.Mutex
	running     bool
	done, total int64
	err         string
}

// fetches are each workspace server's own fetch, so one workspace never
// shows another's.
var fetches sync.Map // *Server -> *fetchState

func (s *Server) fetching() *fetchState {
	f, _ := fetches.LoadOrStore(s, &fetchState{})
	return f.(*fetchState)
}

// ollamaCard is the fetch offered, or how far it has come; "" when Ollama
// is not here or already has a model.
func (s *Server) ollamaCard(hidden string) template.HTML {
	f := s.fetching()
	f.Lock()
	running, done, total, failed := f.running, f.done, f.total, f.err
	f.Unlock()
	if running {
		how := "starting"
		if total > 0 {
			how = fmt.Sprintf("%d%% of %s", done*100/total, llm.FreeModelWords)
		}
		return template.HTML(`<p role="status">Fetching the free model: ` + template.HTMLEscapeString(how) + `. It runs on this computer once it is here; Check again says how far it has come.</p>`)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if !ollamaWithoutModel(ctx) {
		return ""
	}
	out := `<form method="post" action="/model/ollama" class="sw-stack">` + hidden
	if failed != "" {
		out += `<p>` + template.HTMLEscapeString(failed) + `</p>`
	}
	out += string(s.component("button", map[string]any{"label": "Fetch a free model (" + llm.FreeModelWords + ")", "type": "submit", "variant": "primary"}))
	out += `<p class="sw-small sw-muted">Ollama is here with no model yet. This one runs on this computer; your conversations and notes stay here.</p></form>`
	return template.HTML(out)
}

// ollamaFetch starts fetching the free model, in the background: a few
// gigabytes take minutes, and the page is not held while they come.
func (s *Server) ollamaFetch(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if !ollamaWithoutModel(ctx) {
		s.failed(w, r, "Nothing fetched", fmt.Errorf("a model is fetched only for Ollama with none: choose the one it has, or install Ollama first"), "/")
		return
	}
	f := s.fetching()
	f.Lock()
	if !f.running {
		f.running, f.done, f.total, f.err = true, 0, 0, ""
		go s.fetchFreeModel()
	}
	f.Unlock()
	s.tell(w, r, outcome{Title: "Fetching the free model", Text: llm.FreeModelWords + ", through Ollama. It takes a few minutes; the page says how far it has come."}, "/")
}

func (s *Server) fetchFreeModel() {
	f := s.fetching()
	ctx := context.Background()
	err := llm.OllamaPull(ctx, llm.FreeModel, func(done, total int64) {
		f.Lock()
		f.done, f.total = done, total
		f.Unlock()
	})
	if err == nil {
		err = s.useOllama(ctx, llm.FreeModel)
	}
	f.Lock()
	f.running = false
	if err != nil {
		f.err = err.Error()
	}
	f.Unlock()
	s.forgetModel()
}

// useOllama makes an Ollama model the assistant's, through Sameway's copy
// of it with room for the prompt.
func (s *Server) useOllama(ctx context.Context, model string) error {
	name, err := llm.OllamaRoomy(ctx, model)
	if err != nil {
		return err
	}
	if s.app.Chat.SetSetting == nil {
		return fmt.Errorf("this workspace has no settings file")
	}
	for _, kv := range [][2]string{{"llm.base_url", llm.OllamaURL + "/v1"}, {"llm.model", name}, {"llm.provider", "openai"}} {
		if err := s.app.Chat.SetSetting(kv[0], kv[1]); err != nil {
			return err
		}
	}
	return nil
}

// setting is the value a choice's settings give a key, or "".
func setting(kvs [][2]string, key string) string {
	for _, kv := range kvs {
		if kv[0] == key {
			return kv[1]
		}
	}
	return ""
}

// ollamaWithoutModel says Ollama is here with no model: when the free one
// is offered, and the only time it is fetched.
func ollamaWithoutModel(ctx context.Context) bool {
	return llm.OllamaRunning(ctx) && len(llm.Detect(ctx, []llm.Candidate{{Server: "Ollama", BaseURL: llm.OllamaURL + "/v1"}})) == 0
}
