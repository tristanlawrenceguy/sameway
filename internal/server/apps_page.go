package server

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/apps"
)

// The AI apps a person already uses (Claude Desktop, Cursor, VS Code) could
// use their workspace only through a command typed in a terminal, which a
// person who never opens one will not. Now this page does it: an app found
// on this computer is one press, and says the one thing to do after in that
// app's own way; one not found is its own story, closed until opened; the
// apps that live on the internet say plainly why not yet, and what instead.

// appInstall is where each app is had, for its story.
var appInstall = map[string]string{
	"claude-desktop": "https://claude.ai/download",
	"claude-code":    "https://claude.com/claude-code",
	"cursor":         "https://cursor.com",
	"windsurf":       "https://windsurf.com",
	"vscode":         "https://code.visualstudio.com",
	"codex":          "https://developers.openai.com/codex",
}

func appNames() []string {
	var out []string
	for k := range apps.Apps {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return apps.Apps[out[i]].Name < apps.Apps[out[j]].Name })
	return out
}

func (s *Server) appsPage(w http.ResponseWriter, r *http.Request) {
	esc := template.HTMLEscapeString
	ws := s.app.Workspace.Dir
	var found, elsewhere strings.Builder
	for _, key := range appNames() {
		a := apps.Apps[key]
		if !a.Here() {
			steps := `<ol class="sw-stack"><li>` + `<a class="sw-link" href="` + appInstall[key] + `">Get ` + esc(a.Name) + `</a> and install it.</li><li>Come back to this page: it is found, and connecting is a press.</li></ol>`
			if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": a.Name}, template.HTML(steps)); err == nil {
				elsewhere.WriteString(string(body))
			}
			continue
		}
		found.WriteString(`<li class="sw-stack"><strong>` + esc(a.Name) + `</strong>`)
		if apps.Connected(a, ws) {
			found.WriteString(`<p>Connected. ` + esc(a.Note) + `</p>`)
		} else {
			found.WriteString(`<form method="post" action="/apps/connect"><input type="hidden" name="app" value="` + key + `">` +
				string(s.component("button", map[string]any{"label": "Connect " + a.Name, "type": "submit"})) + `</form>`)
		}
		found.WriteString(`</li>`)
	}
	var b strings.Builder
	if found.Len() > 0 {
		b.WriteString(`<h2>On this computer</h2><ul class="sw-plain sw-rows">` + found.String() + `</ul>`)
	}
	if elsewhere.Len() > 0 {
		b.WriteString(`<h2>Another app</h2>` + elsewhere.String())
	}
	web := `<p>ChatGPT, and Claude in a browser or on a phone, run on their makers' computers, not yours, so they can only reach Sameway over the internet, which it does not open on its own. Claude Desktop, above, is the same Claude on this computer and needs nothing more. To share something with ChatGPT or a phone app now, use <a class="sw-link" href="/share">Save to Sameway</a> the other way round: copy what it said into a note.</p>`
	if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "ChatGPT, or Claude in a browser or on a phone"}, template.HTML(web)); err == nil {
		b.WriteString(string(body))
	}
	s.page(w, r, "Your AI apps", template.HTML(b.String()), pageOptions{Lede: "Ask the AI app you already use about your lists and notes, and have it change them, as the assistant here does."})
}

func (s *Server) appsConnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	a, ok := apps.Apps[r.PostForm.Get("app")]
	if !ok {
		s.failed(w, r, "Not connected", errors.New("that is not an app Sameway knows"), "/apps")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "sameway"
	}
	ws := s.app.Workspace.Dir
	if err := apps.Write(a.Path(ws), a, apps.Server(a, exe, ws)); err != nil {
		s.failed(w, r, "Not connected", err, "/apps")
		return
	}
	s.tellAt(w, r, outcome{Title: a.Name + " is connected", Text: a.Note}, "/apps")
}

// appsLine is Your AI apps on Help.
func appsLine() string {
	return `AI apps: Claude Desktop, Cursor, VS Code and others can use this workspace too, a press each, on <a class="sw-link" href="/apps">Your AI apps</a>.`
}
