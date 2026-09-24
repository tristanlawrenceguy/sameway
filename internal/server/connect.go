package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant is how most of Sameway is done, and it needs an AI model
// to think with. On a first run there often is none it can reach: the
// starter points at Ollama, which most people do not have running. The
// first message used to come back as a connection error, and the only fix
// was editing workspace.yaml and starting again; the assistant could not
// help, being the thing that was down.
//
// Now the conversation says, in plain words, that the assistant cannot
// reach a model and why, and offers what is on this computer: a model
// server that answers, Claude Code, a Claude key already saved. Each is
// one press, and says where the conversation would go. Pressing one is
// the person choosing it, so it is asked and answered at once. When
// nothing is found, it says what to install, and Check again looks anew.

// A modelChoice is one way to give the assistant a model, found here.
type modelChoice struct {
	ID       string
	Label    string
	Where    string
	Settings [][2]string
}

// modelState is what was last found, kept a little while so pages stay
// quick: a probe is at most a second and a half, once.
type modelState struct {
	mu      sync.Mutex
	at      time.Time
	ok      bool
	why     string
	choices []modelChoice
}

const modelStateFor = 15 * time.Second

// modelProblem says why the assistant cannot reach a model now, or ""
// when it can, with what could be used instead.
func (s *Server) modelProblem() (string, []modelChoice) {
	s.model.mu.Lock()
	defer s.model.mu.Unlock()
	if time.Since(s.model.at) < modelStateFor {
		if s.model.ok {
			return "", nil
		}
		return s.model.why, s.model.choices
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var ok bool
	var why string
	switch err := s.app.Chat.ProviderErr; {
	case s.app.Chat.Provider == nil && (err == nil || errors.Is(err, llm.ErrNotConfigured)):
		why = "No AI model is set up yet."
	case s.app.Chat.Provider == nil:
		why = "The AI model is not set up right (" + err.Error() + ")."
	default:
		ok, why = llm.Answers(ctx, s.app.Workspace.Config.LLM)
	}
	s.model.at, s.model.ok, s.model.why, s.model.choices = time.Now(), ok, why, nil
	if !ok {
		s.model.choices = modelChoices(ctx)
	}
	return s.model.why, s.model.choices
}

// forgetModel makes the next page look again.
func (s *Server) forgetModel() {
	s.model.mu.Lock()
	s.model.at = time.Time{}
	s.model.mu.Unlock()
}

// modelChoices is every way to a model found on this computer, the ones
// that keep the conversation here first.
func modelChoices(ctx context.Context) []modelChoice {
	var out []modelChoice
	for _, d := range llm.Detect(ctx, llm.DefaultCandidates) {
		out = append(out, modelChoice{
			ID:       "server " + d.BaseURL,
			Label:    fmt.Sprintf("Use %s (%s)", d.Server, d.Model),
			Where:    "Runs on this computer. Your conversations and notes stay here.",
			Settings: [][2]string{{"llm.base_url", d.BaseURL}, {"llm.model", d.Model}, {"llm.provider", "openai"}},
		})
	}
	if _, err := exec.LookPath("claude"); err == nil {
		out = append(out, modelChoice{
			ID:       "claude-code",
			Label:    "Use Claude Code",
			Where:    "Uses your own Claude sign-in, with no key to set up. Your conversations and notes go to Anthropic.",
			Settings: [][2]string{{"llm.model", "sonnet"}, {"llm.provider", "claude-code"}},
		})
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		out = append(out, modelChoice{
			ID:       "anthropic",
			Label:    "Use Claude with your saved key",
			Where:    "Uses the Claude key saved on this computer. Your conversations and notes go to Anthropic.",
			Settings: [][2]string{{"llm.api_key_env", "ANTHROPIC_API_KEY"}, {"llm.model", "claude-sonnet-5"}, {"llm.provider", "anthropic"}},
		})
	}
	return out
}

// connectCard is the conversation saying the assistant cannot reach a
// model, and what the person can do about it here and now.
func (s *Server) connectCard(from string) template.HTML {
	why, choices := s.modelProblem()
	if why == "" {
		return ""
	}
	esc := template.HTMLEscapeString
	hidden := `<input type="hidden" name="from" value="` + esc(from) + `">`
	var b strings.Builder
	b.WriteString(`<section class="sw-connect sw-stack" aria-label="Connect the assistant">`)
	// Said as an alert, so it is heard when the page opens, not found later.
	b.WriteString(string(s.component("alert", map[string]any{"kind": "info", "title": "Connect the assistant to an AI model",
		"message": why + " The assistant needs an AI model to think with. Everything else in Sameway works without one."})))
	if len(choices) > 0 {
		b.WriteString(`<p>Found on this computer:</p><ul class="sw-plain sw-stack sw-connect__choices">`)
		for _, c := range choices {
			b.WriteString(`<li><form method="post" action="/model/use">` + hidden + `<input type="hidden" name="choice" value="` + esc(c.ID) + `">`)
			b.WriteString(string(s.component("button", map[string]any{"label": c.Label, "type": "submit", "variant": "primary"})))
			b.WriteString(`</form><p class="sw-small sw-muted">` + esc(c.Where) + `</p></li>`)
		}
		b.WriteString(`</ul>`)
	} else {
		b.WriteString(`<p>Nothing was found on this computer yet. Either of these works:</p><ul class="sw-connect__ways">`)
		b.WriteString(`<li><a class="sw-link" href="https://ollama.com/download">Ollama</a> runs AI models on this computer, free, and nothing leaves it. Install it, then in a terminal run <code>ollama pull llama3.1</code>.</li>`)
		b.WriteString(`<li><a class="sw-link" href="https://claude.com/claude-code">Claude Code</a> uses your Claude account. Install it and sign in.</li>`)
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<form method="post" action="/model/check">` + hidden)
	b.WriteString(string(s.component("button", map[string]any{"label": "Check again", "type": "submit", "variant": "secondary"})))
	b.WriteString(`</form></section>`)
	return template.HTML(b.String())
}

// modelUse is the person pressing one of the choices: the model is set,
// and the next message goes to it.
func (s *Server) modelUse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	want := r.PostForm.Get("choice")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	for _, c := range modelChoices(ctx) {
		if c.ID != want {
			continue
		}
		if s.app.Chat.SetSetting == nil {
			s.failed(w, r, "Not connected", errors.New("this workspace has no settings file"), "/")
			return
		}
		for _, kv := range c.Settings {
			if err := s.app.Chat.SetSetting(kv[0], kv[1]); err != nil {
				s.failed(w, r, "Not connected", err, "/")
				return
			}
		}
		s.forgetModel()
		s.tell(w, r, outcome{Title: "Connected", Text: strings.TrimPrefix(c.Label, "Use ") + " is the assistant's model now. Say hello."}, "/")
		return
	}
	s.forgetModel()
	s.failed(w, r, "Not connected", errors.New("that is no longer on this computer; the choices have been looked for again"), "/")
}

// modelCheck looks again, after the person installed or started something.
func (s *Server) modelCheck(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.forgetModel()
	if why, _ := s.modelProblem(); why == "" {
		s.tell(w, r, outcome{Title: "The assistant can reach its model", Text: "Say hello."}, "/")
		return
	}
	http.Redirect(w, r, backOf(r, "/"), http.StatusSeeOther)
}
