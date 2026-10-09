package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
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
	// woke is when Sameway last started Ollama itself (ollama_wake.go);
	// waking says it is starting now, so nothing else is offered.
	woke   time.Time
	waking bool
}

const modelStateFor = 15 * time.Second

// modelProblem says why the assistant cannot reach a model now, or ""
// when it can, with what could be used instead.
func (s *Server) modelProblem() (string, []modelChoice) {
	s.model.mu.Lock()
	defer s.model.mu.Unlock()
	fresh := modelStateFor
	if s.model.waking {
		fresh = 3 * time.Second // Ollama starting is seen as soon as it answers
	}
	if time.Since(s.model.at) < fresh {
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
		ok, why = llm.Answers(ctx, s.app.LLMConfig())
		why = s.wakeOllama(ok, why)
	}
	s.model.at, s.model.ok, s.model.why, s.model.choices = time.Now(), ok, why, nil
	if !ok {
		s.model.choices = s.modelChoices(ctx)
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
func (s *Server) modelChoices(ctx context.Context) []modelChoice {
	var out []modelChoice
	for _, d := range llm.Detect(ctx, llm.DefaultCandidates) {
		out = append(out, modelChoice{
			ID:       "server " + d.BaseURL,
			Label:    fmt.Sprintf("Use %s (%s)", d.Server, d.Model),
			Where:    "Runs on this computer, free. Your conversations and notes stay here. Slower than Claude, and it gets more wrong.",
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
	if s.keys().Get("ANTHROPIC_API_KEY") != "" {
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
	var b strings.Builder
	b.WriteString(`<div class="sw-connect sw-stack" data-wait="` + esc(s.modelWait()) + `"><h2 class="sw-visually-hidden">Connect the assistant</h2>`)
	// A status message, and a heading to find it by from the page's outline.
	b.WriteString(string(s.part(ui.Alert{Kind: ui.Info, Title: "Connect the assistant to an AI model",
		Message: why + " The assistant needs an AI model to think with. Everything else in Sameway works without one."})))
	if len(choices) > 0 {
		b.WriteString(`<p>Found on this computer:</p><ul class="sw-plain sw-stack sw-connect__choices">`)
		for _, c := range choices {
			b.WriteString(`<li>` + string(s.form(ui.Form{Action: "/model/use", From: from, Hidden: ui.Hidden("choice", c.ID), Button: &ui.Button{Label: c.Label, Variant: ui.Primary}})))
			b.WriteString(`<p class="sw-small sw-muted">` + esc(c.Where) + `</p></li>`)
		}
		b.WriteString(`</ul>`)
	}
	if !s.model.waking {
		if len(choices) > 0 {
			b.WriteString(`<p>Or something else:</p>`)
		}
		b.WriteString(s.modelStoriesHTML(from)) // connect_stories.go
	}
	b.WriteString(string(s.ollamaCard(from))) // ollama_setup.go
	b.WriteString(string(s.form(ui.Form{Action: "/model/check", From: from, Button: &ui.Button{Label: "Check again", Variant: ui.Secondary}})))
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// modelUse is the person pressing one of the choices: the model is set,
// and the next message goes to it.
func (s *Server) modelUse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	want := r.PostForm.Get("choice")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	for _, c := range s.modelChoices(ctx) {
		if c.ID != want {
			continue
		}
		if s.app.Records.SetSetting == nil {
			s.failed(w, r, "Not connected", errors.New("this workspace has no settings file"), "/")
			return
		}
		// An Ollama model is used through Sameway's copy of it, with room
		// for the prompt Ollama's default would cut (ollama_setup.go).
		if base, model := setting(c.Settings, "llm.base_url"), setting(c.Settings, "llm.model"); llm.IsOllama(base) {
			slow, done := context.WithTimeout(r.Context(), time.Minute)
			defer done()
			if err := s.useOllama(slow, model); err != nil {
				s.failed(w, r, "Not connected", err, "/")
				return
			}
			s.forgetModel()
			s.tell(w, r, outcome{Title: "Connected", Text: strings.TrimPrefix(c.Label, "Use ") + " is the assistant's model now. Say hello."}, "/")
			return
		}
		for _, kv := range c.Settings {
			if err := s.app.Records.SetSetting(kv[0], kv[1]); err != nil {
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
	http.Redirect(w, r, web.BackOf(r, "/"), http.StatusSeeOther)
}

// ollamaDownload is Ollama's installer for this computer, so the person
// downloads it with one press instead of finding it on a page of them.
// Linux installs it with a command, which the page there gives.
func ollamaDownload() string {
	switch runtime.GOOS {
	case "windows":
		return "https://ollama.com/download/OllamaSetup.exe"
	case "darwin":
		return "https://ollama.com/download/Ollama.dmg"
	}
	return "https://ollama.com/download/linux"
}

// modelWait is the connect card in a few words that change when it would:
// what was found, and how far a fetch has come. The page asks for them
// while the card is up and follows when they change (37-connect-wait.js),
// so Ollama installed, or a model fetched, shows without Check again.
func (s *Server) modelWait() string {
	why, choices := s.modelProblem()
	if why == "" {
		return "ready"
	}
	parts := []string{why}
	for _, c := range choices {
		parts = append(parts, c.ID)
	}
	f := s.fetching()
	f.Lock()
	if f.running {
		parts = append(parts, "fetching", fmt.Sprint(percent(f.done, f.total)))
	}
	parts = append(parts, f.err)
	f.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	parts = append(parts, fmt.Sprint(ollamaWithoutModel(ctx)))
	return strings.Join(parts, "|")
}

func percent(done, total int64) int64 {
	if total <= 0 {
		return 0
	}
	return done * 100 / total
}

// modelWaitState is modelWait for the page that is waiting.
func (s *Server) modelWaitState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, s.modelWait())
}

// wakeOllama starts Ollama when the workspace uses it and it is installed
// but not answering, at most once a minute, and says so in place of an
// address that is not answering. Called with s.model held.
func (s *Server) wakeOllama(ok bool, why string) string {
	s.model.waking = false
	if ok || !llm.IsOllama(s.app.Workspace.Config.LLM.BaseURL) {
		return why
	}
	if time.Since(s.model.woke) < time.Minute {
		s.model.waking = true
		return "Ollama is starting. This page follows when it is ready."
	}
	if llm.WakeOllama() {
		s.model.woke, s.model.waking = time.Now(), true
		return "Ollama was not running, so Sameway is starting it. This page follows when it is ready."
	}
	return "Ollama is not running on this computer."
}
