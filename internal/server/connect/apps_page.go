package connect

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/apps"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// The AI apps a person already uses (Claude Desktop, Cursor, VS Code) could
// use their workspace only through a command typed in a terminal, which a
// person who never opens one will not. Now this page does it: an app found
// on this computer is one press, and says the one thing to do after in that
// app's own way; one not found is its own story, closed until opened; the
// apps that live on the internet say plainly why not yet, and what instead.
// What a connected app may do is said once, before the first press, and
// taking it back is one press too (design/foundations/connecting.md).

// appInstall is where each app is had, for its story.
var appInstall = map[string]string{
	"claude-desktop": "https://claude.ai/download",
	"claude-code":    "https://claude.com/claude-code",
	"cursor":         "https://cursor.com",
	"windsurf":       "https://windsurf.com",
	"vscode":         "https://code.visualstudio.com",
	"codex":          "https://developers.openai.com/codex",
}

// appsMay is what a connected app may do, said once above them all.
const appsMay = `<p>An app you connect can read and change everything in this workspace, as you can, while it is open on this computer. What it changes is in <a class="sw-link" href="/activity">Activity</a>, where each change can be undone. Disconnect it here whenever you like.</p>`

func appNames() []string {
	var out []string
	for k := range apps.Apps {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return apps.Apps[out[i]].Name < apps.Apps[out[j]].Name })
	return out
}

func (s *Service) appsPage(w http.ResponseWriter, r *http.Request) {
	var found, elsewhere strings.Builder
	for _, key := range appNames() {
		a := apps.Apps[key]
		if !a.Here() {
			steps := `<ol class="sw-stack"><li>` + `<a class="sw-link" href="` + appInstall[key] + `">Get ` + template.HTMLEscapeString(a.Name) + `</a> and install it.</li><li>Come back to this page: it is found, and connecting is a press.</li></ol>`
			if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": a.Name}, template.HTML(steps)); err == nil {
				elsewhere.WriteString(string(body))
			}
			continue
		}
		found.WriteString(s.appRow(key, a))
	}
	var b strings.Builder
	if found.Len() > 0 {
		b.WriteString(`<h2>On this computer</h2>` + appsMay + `<ul class="sw-plain sw-rows">` + found.String() + `</ul>`)
	}
	if elsewhere.Len() > 0 {
		b.WriteString(`<h2>Another app</h2>` + elsewhere.String())
	}
	online := `<p>ChatGPT, and Claude in a browser or on a phone, run on their makers' computers, not yours, so they can only reach Sameway over the internet, which it does not open on its own. Claude Desktop, above, is the same Claude on this computer and needs nothing more. To share something with ChatGPT or a phone app now, use <a class="sw-link" href="/share">Save to Sameway</a> the other way round: copy what it said into a note.</p>`
	if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "ChatGPT, or Claude in a browser or on a phone"}, template.HTML(online)); err == nil {
		b.WriteString(string(body))
	}
	b.WriteString(s.agentsHTML()) // apps_agents.go
	s.Page(w, r, "Your AI apps", template.HTML(b.String()), web.PageOptions{Lede: "Ask the AI app you already use about your lists and notes, and have it change them, as the assistant here does."})
}

// appRow is one app found on this computer: whether it reaches this
// workspace, the press that changes that, and exactly what it starts.
func (s *Service) appRow(key string, a apps.App) string {
	esc := template.HTMLEscapeString
	ws := s.app.Workspace.Dir
	var b strings.Builder
	b.WriteString(`<li class="sw-stack"><strong>` + esc(a.Name) + `</strong>`)
	connect := &ui.Button{Label: "Connect " + a.Name}
	switch entry := apps.Entry(a, ws); {
	case apps.Same(entry, ws):
		b.WriteString(`<p>Connected. ` + esc(a.Note) + `</p>`)
		b.WriteString(string(s.Form(ui.Form{Action: "/apps/disconnect", Hidden: ui.Hidden("app", key),
			Button: &ui.Button{Label: "Disconnect " + a.Name, Variant: ui.Secondary}})))
	case entry != "":
		b.WriteString(`<p>Connected to another workspace, ` + esc(filepath.Base(entry)) + `. Connecting it here moves it to this one.</p>`)
		connect.Label = "Connect " + a.Name + " here"
		fallthrough
	default:
		b.WriteString(string(s.Form(ui.Form{Action: "/apps/connect", Hidden: ui.Hidden("app", key), Button: connect})))
	}
	// What one press writes and runs, whole, so it can be checked before
	// (MCP security best practices, local server consent).
	exactly := `<div class="sw-prose sw-stack"><p>Connecting adds Sameway to <code>` + esc(a.Path(ws)) + `</code>, keeping everything else in it. ` + esc(a.Name) + ` then starts this command on this computer:</p><p><code>` + esc(command(a, ws)) + `</code></p></div>`
	if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "What " + a.Name + " starts"}, template.HTML(exactly)); err == nil {
		b.WriteString(string(body))
	}
	b.WriteString(`</li>`)
	return b.String()
}

// command is the line an app runs to start Sameway for this workspace.
func command(a apps.App, ws string) string {
	server := apps.Server(a, executable(), ws)
	words := []string{quote(server["command"].(string))}
	for _, arg := range server["args"].([]string) {
		words = append(words, quote(arg))
	}
	return strings.Join(words, " ")
}

func quote(s string) string {
	if strings.ContainsAny(s, " \t\"") {
		return `"` + s + `"`
	}
	return s
}

func executable() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "sameway"
}

func (s *Service) appsConnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	a, ok := apps.Apps[r.PostForm.Get("app")]
	if !ok {
		s.Failed(w, r, "Not connected", errors.New("that is not an app Sameway knows"), "/apps")
		return
	}
	ws := s.app.Workspace.Dir
	if err := apps.Write(a.Path(ws), a, apps.Server(a, executable(), ws)); err != nil {
		s.Failed(w, r, "Not connected", err, "/apps")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: a.Name + " is connected", Text: a.Note}, "/apps")
}

// appsDisconnect takes Sameway out of the app's file, as easily as it went
// in; the app lets go of it when it next starts.
func (s *Service) appsDisconnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	a, ok := apps.Apps[r.PostForm.Get("app")]
	if !ok {
		s.Failed(w, r, "Not disconnected", errors.New("that is not an app Sameway knows"), "/apps")
		return
	}
	// Another workspace's entry in the same file is that workspace's to take.
	if ws := s.app.Workspace.Dir; !apps.Same(apps.Entry(a, ws), ws) {
		s.TellAt(w, r, web.Outcome{Title: a.Name + " is not connected", Text: a.Name + " does not start Sameway for this workspace, so there was nothing to take out."}, "/apps")
		return
	}
	if err := apps.Remove(a.Path(s.app.Workspace.Dir), a); err != nil {
		s.Failed(w, r, "Not disconnected", err, "/apps")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: a.Name + " is disconnected", Text: a.Name + " no longer starts Sameway. Once it is closed and opened again, it cannot reach this workspace. Connect it here again whenever you like."}, "/apps")
}

// AppsLine is Your AI apps on Help.
func AppsLine() string {
	return `AI apps: Claude Desktop, Cursor, VS Code and others can use this workspace too, a press each, on <a class="sw-link" href="/apps">Your AI apps</a>.`
}
