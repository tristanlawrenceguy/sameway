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
	"github.com/tristanlawrenceguy/sameway/internal/ui"
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
	// done and total are how far a fetch has come, in bytes.
	done, total int64
	err         string
	// read says the records have been read once, by the model there is now.
	read bool
}

// meaningKey is where the workspace's store keeps its meaningState, so two
// workspaces open side by side never share one, and it goes with the store.
type meaningKey struct{}

func (s *Server) meaningState() *meaningState {
	return s.app.Store.Keep(meaningKey{}, func() any { return &meaningState{} }).(*meaningState)
}

// embedModel is an embedding model Ollama has, or "".
func embedModel(ctx context.Context, ollama llm.Ollama) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollama.Base()+"/api/tags", nil)
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
	ollama := s.app.Options().Ollama
	model := embedModel(ctx, ollama)
	if model == "" {
		search.UseMeaning(s.app.Store, nil)
		return
	}
	st.Lock()
	if st.idx == nil || st.idx.Model != model {
		st.idx = &meaning.Index{Base: ollama.Base(), Model: model, Keep: s.app.Store}
		st.read = false
	}
	idx := st.idx
	st.Unlock()
	docs := search.Docs(s.app.Store, s.app.Types, s.reader())
	var in []meaning.Doc
	for _, d := range docs {
		in = append(in, meaning.Doc{Key: d.Key, Text: d.Text})
	}
	_, err := idx.Refresh(ctx, in)
	st.Lock()
	st.err, st.read = "", err == nil
	if err != nil {
		st.err = err.Error()
	}
	st.Unlock()
	if err != nil {
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

// meaningLine is search by meaning on Help, for the owner: being fetched
// and how far, reading the records, on, not working and why, offered, or
// nothing when there is no Ollama. Help is read when it is opened, so each
// state is said as it is then, once; nothing changes in place.
func (s *Server) meaningLine() string {
	st := s.meaningState()
	st.Lock()
	fetching, done, total, failed := st.fetching, st.done, st.total, st.err
	ready := st.read
	st.Unlock()
	if fetching {
		how := meaning.FetchWords
		if total > 0 {
			how = fmt.Sprintf("%d%% of %s", done*100/total, meaning.FetchWords)
		}
		return "Search by meaning: fetching " + how + ". Searches find by meaning a few minutes after it is here."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ollama := s.app.Options().Ollama
	switch on := embedModel(ctx, ollama) != ""; {
	case on && failed != "":
		return "Search by meaning: not working, so searches find by words only. The last try said: " + template.HTMLEscapeString(failed) + ". It tries again every few minutes."
	case on && !ready:
		return "Search by meaning: reading what you have, on this computer. Searches find by meaning once that is done."
	case on:
		return "Search by meaning: on. Searching finds what something is about, not only its words, on this computer."
	case !ollama.Running(ctx):
		return ""
	}
	out := `Search by meaning: off. With it, searching for the plumber finds the note about the boiler repair. It is a model for Ollama (` + meaning.FetchWords + `), and nothing leaves this computer.`
	if failed != "" {
		out += " The last try said: " + template.HTMLEscapeString(failed) + "."
	}
	return out + " " + string(s.form(ui.Form{Action: "/meaning/fetch", Button: &ui.Button{Label: "Turn on search by meaning", Variant: ui.Secondary}}))
}

func (s *Server) meaningFetch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ollama := s.app.Options().Ollama
	if !ollama.Running(ctx) {
		s.failed(w, r, "Not fetched", errors.New("search by meaning needs Ollama running on this computer"), "/help")
		return
	}
	st := s.meaningState()
	st.Lock()
	if !st.fetching {
		st.fetching, st.done, st.total, st.err = true, 0, 0, ""
		go func() {
			err := ollama.Pull(context.Background(), meaning.FetchModel, func(done, total int64) {
				st.Lock()
				st.done, st.total = done, total
				st.Unlock()
			})
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
	s.tell(w, r, outcome{Title: "Fetching search by meaning", Text: fmt.Sprintf("%s, through Ollama. Help says how far it has come.", meaning.FetchWords)}, "/help")
}
