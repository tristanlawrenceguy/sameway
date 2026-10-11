package connect

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An agent let in with a key of its own (sameway agent add) could be seen
// and taken away only from a terminal, or by asking the assistant. A
// person sees every app that reaches their account in one place, with
// what each may do and when it was last used, and takes one away there
// (GitHub's Authorized apps, Google's third-party access). So the agents
// with keys are listed here, beside the apps, when there are any. Keys
// are still made only at the command line, where one is shown once.

// mayWords is what an agent's access lets it do, in plain words.
var mayWords = map[string]string{
	records.View:  "May look at everything, and change nothing.",
	records.Edit:  "May look at and change your lists and notes.",
	records.Owner: "May do everything you can.",
}

// agentsHTML is the agents with keys, each with the press that takes its
// key away, or nothing when there are none.
func (s *Service) agentsHTML() string {
	agents := records.Agents(s.app.Store)
	if len(agents) == 0 {
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<h2>Agents with a key</h2><p>Each reaches this workspace with a key of its own. Taking a key away stops that agent at once; undoing it in <a class="sw-link" href="/activity">Activity</a> lets it back in.</p><ul class="sw-plain sw-rows">`)
	for _, a := range agents {
		name, _ := a.Fields["name"].(string)
		access, _ := a.Fields["access"].(string)
		used := "Never used."
		if at, err := time.Parse(time.RFC3339, str(a.Fields["last_used"])); err == nil {
			used = "Last used " + lowerFirst(when.Ago(at, s.Now(), s.app.Workspace.Hours24())) + "."
		}
		b.WriteString(`<li class="sw-stack"><strong>` + esc(name) + `</strong><p>` + esc(mayWords[access]) + ` ` + esc(used) + `</p>`)
		b.WriteString(string(s.Form(ui.Form{Action: "/apps/agents/remove", Hidden: ui.Hidden("name", name),
			Button: &ui.Button{Label: "Take its key away", Context: name, Variant: ui.Secondary}})))
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

// agentRemove takes an agent's key away, logged so it can be undone.
func (s *Service) agentRemove(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := strings.TrimSpace(r.PostForm.Get("name"))
	ch, err := records.TakeAgentAway(s.app.Store, name)
	if err != nil {
		s.Failed(w, r, "Key not taken away", err, "/apps")
		return
	}
	undo := s.Record(r, ch)
	s.Changed()
	s.TellAt(w, r, web.Outcome{Title: name + "'s key is taken away", Text: name + " can no longer reach this workspace.", Undo: undo, Of: name + "'s key"}, "/apps")
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// lowerFirst is "Today at 2pm" inside a sentence: "today at 2pm". A date
// ("2 Oct") is left as it is.
func lowerFirst(s string) string {
	if strings.HasPrefix(s, "Today") || strings.HasPrefix(s, "Yesterday") {
		return strings.ToLower(s[:1]) + s[1:]
	}
	return s
}
