package server

import (
	"html/template"
	"net/http"
	"runtime"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// When something in Sameway did not work, a person had nowhere to say so
// but a GitHub page they would have to find, and the makers learned only
// what they guessed. Help's Something not right? takes what happened in
// the person's words and shows, before anything leaves, all that would
// go: those words, Sameway's version, the system, the kind of model (never
// a key or an address) and the last few failures in the log, as the log
// says them. It opens the makers' new-issue page with it filled in, where
// the person reads it, changes it and sends it themselves, or not:
// Sameway sends nothing.

// issuesNew is where the makers take reports.
const issuesNew = "https://github.com/tristanlawrenceguy/sameway/issues/new"

// feedbackFacts is what goes with the person's words: nothing that names
// them, their records, a key or an address.
func (s *Server) feedbackFacts() string {
	var b strings.Builder
	b.WriteString("Sameway " + update.Version + " on " + runtime.GOOS + "/" + runtime.GOARCH)
	if p := s.app.Workspace.Config.LLM.Provider; p != "" {
		b.WriteString(", model: " + p)
	}
	recent, _ := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 200})
	var failed []string
	for _, r := range recent {
		if r.Fields["action"] == "failed" && len(failed) < 5 {
			failed = append(failed, "- "+r.CreatedAt.Format("2 Jan 15:04")+": "+chat.SanitizeError(chat.Sentence(s.app.Store, r.Fields)))
		}
	}
	if len(failed) > 0 {
		b.WriteString("\n\nRecent failures:\n" + strings.Join(failed, "\n"))
	}
	return b.String()
}

// feedbackSection is Something not right? on Help, for the owner.
func (s *Server) feedbackSection() string {
	return `<section class="sw-stack" aria-labelledby="help-tell"><h2 id="help-tell">Something not right?</h2>
<p>Tell the makers what happened or what you wished it did. You see everything that would go before it goes, and you send it yourself.</p>
<form method="post" action="/feedback" class="sw-stack">` +
		string(s.component("textarea", map[string]any{"label": "What happened", "name": "what", "rows": 4, "required": true, "hint": "What you did, what you expected, and what Sameway did instead."})) +
		string(s.component("button", map[string]any{"label": "See what would go", "type": "submit", "variant": "secondary"})) + `</form></section>`
}

// feedback shows what would go, in a form that opens the makers' page
// with it, which the person sends there or not.
func (s *Server) feedback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	what := strings.TrimSpace(r.PostForm.Get("what"))
	title := what
	if i := strings.IndexAny(title, ".\n"); i > 0 {
		title = title[:i]
	}
	if rs := []rune(title); len(rs) > 80 {
		title = string(rs[:80]) + "…"
	}
	body := what + "\n\n---\n" + s.feedbackFacts()
	esc := template.HTMLEscapeString
	page := `<p>This is all that would go, to the makers' page on GitHub, where you can change it and send it, which needs a GitHub account. Nothing has been sent.</p>
<form method="get" action="` + issuesNew + `" target="_blank" rel="noopener" class="sw-stack">` +
		string(s.component("text-field", map[string]any{"label": "Title", "name": "title", "value": title})) +
		`<label for="feedback-body">What goes</label><textarea id="feedback-body" name="body" rows="14" class="sw-textarea">` + esc(body) + `</textarea>` +
		string(s.component("button", map[string]any{"label": "Open it on GitHub", "type": "submit"})) + `</form>`
	s.page(w, r, "Tell the makers", template.HTML(page), pageOptions{})
}
