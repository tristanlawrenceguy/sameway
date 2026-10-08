package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person with no model on their computer could only install one and run
// a command in a terminal, or install a developer's tool and sign in. Most
// people can do neither. Anyone can make a key on a website and paste it:
// a key from Anthropic is Claude, and one from OpenRouter reaches many
// models, some of them free. The key is kept in the person's own settings
// folder (llm.SaveKey), never the workspace, and the page says where the
// conversation then goes.

// keyKinds are the keys a person can paste, by how each begins.
var keyKinds = []struct {
	prefix, env, label, company, page string
	settings                          [][2]string
}{
	{"sk-ant-", "ANTHROPIC_API_KEY", "Claude (Anthropic)", "Anthropic", "console.anthropic.com/settings/keys",
		[][2]string{{"llm.api_key_env", "ANTHROPIC_API_KEY"}, {"llm.model", "claude-sonnet-5"}, {"llm.provider", "anthropic"}}},
	{"sk-or-", "OPENROUTER_API_KEY", "OpenRouter", "OpenRouter", "openrouter.ai/keys",
		[][2]string{{"llm.api_key_env", "OPENROUTER_API_KEY"}, {"llm.base_url", "https://openrouter.ai/api/v1"}, {"llm.model", "openrouter/auto"}, {"llm.provider", "openai"}}},
}

func (s *Server) modelRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /model/use", s.modelUse)
	m.HandleFunc("POST /model/check", s.modelCheck)
	m.HandleFunc("GET /model/wait", s.modelWaitState) // connect.go
	m.HandleFunc("POST /model/key", s.modelKey)
	m.HandleFunc("POST /model/ollama", s.ollamaFetch) // ollama_setup.go
}

// keyForm is the paste-a-key part of the connect card.
func (s *Server) keyForm(hidden string) template.HTML {
	var b strings.Builder
	b.WriteString(`<form method="post" action="/model/key" class="sw-stack">` + hidden)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Or paste a key", "name": "key", "type": "password", "autocomplete": "off",
		"hint": "A key from Anthropic (it begins sk-ant-) or OpenRouter (sk-or-). Claude answers quickest and gets the most right; it costs a little for each message. The key is kept in your own settings on this computer, not in the workspace; your conversations then go to that company."})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Use this key", "type": "submit", "variant": "secondary"})))
	b.WriteString(`<p class="sw-small">Make one at <a class="sw-link" href="https://console.anthropic.com/settings/keys">Anthropic's keys page</a> or <a class="sw-link" href="https://openrouter.ai/keys">OpenRouter's</a>.</p>`)
	b.WriteString(`</form>`)
	return template.HTML(b.String())
}

// modelKey is a person pasting a key: kept, and the model set to it.
func (s *Server) modelKey(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := strings.TrimSpace(r.PostForm.Get("key"))
	for _, k := range keyKinds {
		if !strings.HasPrefix(key, k.prefix) {
			continue
		}
		if s.app.Records.SetSetting == nil {
			s.failed(w, r, "Not connected", errors.New("this workspace has no settings file"), "/")
			return
		}
		// Asked about before it is kept: a key the provider does not know is
		// said to be wrong here, not at the first message (llm.CheckKey).
		verdict, why := llm.CheckKey(r.Context(), k.env, key)
		switch verdict {
		case llm.KeyRefused:
			s.failed(w, r, "Not connected", fmt.Errorf("%s did not accept this key. Check that all of it was copied, or make a new one on %s", k.company, k.page), "/")
			return
		case llm.KeyNoCredit:
			s.failed(w, r, "Not connected", fmt.Errorf("this key has no credit left. Add some on %s, then paste it again", k.page), "/")
			return
		}
		if err := llm.SaveKey(k.env, key); err != nil {
			s.failed(w, r, "Not connected", errors.New("the key could not be kept: "+err.Error()), "/")
			return
		}
		for _, kv := range k.settings {
			if err := s.app.Records.SetSetting(kv[0], kv[1]); err != nil {
				s.failed(w, r, "Not connected", err, "/")
				return
			}
		}
		s.forgetModel()
		said := k.label + " is the assistant's model now. Say hello."
		if verdict == llm.KeyUnchecked {
			said += " The key could not be checked just now (" + why.Error() + "); if the first message fails, paste it again."
		}
		s.tell(w, r, outcome{Title: "Connected", Text: said}, "/")
		return
	}
	s.failed(w, r, "Not connected", errors.New("that is not a key Sameway knows: paste one from Anthropic, which begins sk-ant-, or from OpenRouter, which begins sk-or-"), "/")
}
