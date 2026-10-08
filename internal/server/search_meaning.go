package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/meaning"
	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// Search by meaning, for a workspace on a computer with Ollama and an
// embedding model: every few minutes the records whose words changed are
// read again (internal/meaning), and every search, the page's, the
// assistant's and an agent's, also finds what is near in meaning. Help
// offers the model when Ollama is here without one.

type meaningState struct {
	sync.Mutex
	idx      *meaning.Index
	fetching bool
	err      string
}

var meanings sync.Map // *Server -> *meaningState

func (s *Server) meaningState() *meaningState {
	m, _ := meanings.LoadOrStore(s, &meaningState{})
	return m.(*meaningState)
}

// embedModel is an embedding model Ollama has, or "".
func embedModel(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, llm.OllamaURL+"/api/tags", nil)
	if err != nil {
		return ""
	}
	res, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	json.NewDecoder(res.Body).Decode(&tags)
	for _, want := range meaning.Models {
		for _, m := range tags.Models {
			if m.Name == want || strings.HasPrefix(m.Name, want+":") {
				return m.Name
			}
		}
	}
	return ""
}

// KeepMeaning keeps the records' meaning current while ctx lasts.
func (s *Server) KeepMeaning(ctx context.Context) {
	go func() {
		tick := time.NewTicker(2 * time.Minute)
		defer tick.Stop()
		for {
			s.refreshMeaning(ctx)
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// RefreshMeaning reads again what changed and has search use it, and says
// whether search by meaning is on.
func (s *Server) RefreshMeaning(ctx context.Context) bool {
	s.refreshMeaning(ctx)
	_, on := searchByMeaning(s)
	return on
}

func searchByMeaning(s *Server) (*meaning.Index, bool) {
	st := s.meaningState()
	st.Lock()
	defer st.Unlock()
	return st.idx, st.idx != nil && st.idx.Ready() && st.err == ""
}

// refreshMeaning reads again what changed, and has search use it.
func (s *Server) refreshMeaning(ctx context.Context) {
	st := s.meaningState()
	model := embedModel(ctx)
	if model == "" {
		search.UseMeaning(s.app.Store, nil)
		return
	}
	st.Lock()
	if st.idx == nil || st.idx.Model != model {
		st.idx = &meaning.Index{Base: llm.OllamaURL, Model: model, Keep: s.app.Store}
	}
	idx := st.idx
	st.Unlock()
	docs := search.Docs(s.app.Store, s.app.Types)
	var in []meaning.Doc
	for _, d := range docs {
		in = append(in, meaning.Doc{Key: d.Key, Text: d.Text})
	}
	if _, err := idx.Refresh(ctx, in); err != nil {
		st.Lock()
		st.err = err.Error()
		st.Unlock()
		return
	}
	search.UseMeaning(s.app.Store, func(q string) []string {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		found, err := idx.Find(c, q)
		if err != nil {
			return nil
		}
		var keys []string
		for _, n := range found {
			keys = append(keys, n.Key)
		}
		return keys
	})
}

// meaningLine is search by meaning on Help, for the owner: on, offered,
// being fetched, or nothing when there is no Ollama.
func (s *Server) meaningLine() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	st := s.meaningState()
	st.Lock()
	fetching, failed := st.fetching, st.err
	st.Unlock()
	switch {
	case embedModel(ctx) != "":
		return "Search by meaning: on. Searching finds what something is about, not only its words, on this computer."
	case fetching:
		return "Search by meaning: fetching " + meaning.FetchWords + "."
	case !llm.OllamaRunning(ctx):
		return ""
	}
	out := `Search by meaning: off. With it, searching for the plumber finds the note about the boiler repair. It is a model for Ollama (` + meaning.FetchWords + `), and nothing leaves this computer.`
	if failed != "" {
		out += " The last try said: " + template.HTMLEscapeString(failed) + "."
	}
	return out + ` <form method="post" action="/meaning/fetch">` + string(s.component("button", map[string]any{"label": "Turn on search by meaning", "type": "submit", "variant": "secondary"})) + `</form>`
}

func (s *Server) meaningFetch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if !llm.OllamaRunning(ctx) {
		s.failed(w, r, "Not fetched", errors.New("search by meaning needs Ollama running on this computer"), "/help")
		return
	}
	st := s.meaningState()
	st.Lock()
	if !st.fetching {
		st.fetching, st.err = true, ""
		go func() {
			err := llm.OllamaPull(context.Background(), meaning.FetchModel, func(int64, int64) {})
			st.Lock()
			st.fetching = false
			if err != nil {
				st.err = err.Error()
			}
			st.Unlock()
			s.refreshMeaning(context.Background())
		}()
	}
	st.Unlock()
	s.tell(w, r, outcome{Title: "Fetching search by meaning", Text: fmt.Sprintf("%s, through Ollama. Searches find by meaning a few minutes after it is here.", meaning.FetchWords)}, "/help")
}

func (s *Server) meaningRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /meaning/fetch", s.meaningFetch)
}
