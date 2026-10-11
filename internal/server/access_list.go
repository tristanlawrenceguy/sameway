package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Who can open this, on Workspaces: everyone the owner has let in, in one
// list, whichever way they came (an invite on the Wi-Fi, or Tailscale),
// with what each may do in words and a way to take it back. Before, the
// owner saw devices but not people: someone invited and not yet joined
// was nowhere, someone on Tailscale was only in the chat that let them
// in, and taking access from someone invited on the Wi-Fi meant removing
// each of their devices while their invite still worked. Google Drive's
// "People with access" and Notion's share menu put the same list where
// sharing is done, each row with its role and Remove access.

// accessSection is Who can open this: the owner's alone, shown when
// anyone may, or when the owner could invite someone on the Wi-Fi.
func (s *Server) accessSection(r *http.Request) string {
	if !s.chatFor(r).IsOwner() {
		return ""
	}
	people := s.peopleLetIn()
	canInvite := s.fleet != nil && s.fleet.LAN != nil
	if len(people) == 0 && !canInvite {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-access"><h2 id="ws-access">Who can open this</h2>`)
	if len(people) == 0 {
		b.WriteString(`<p>Only you. To let someone in your home or office open it from their own phone or computer, invite them by name under On your phone.</p></section>`)
		return b.String()
	}
	b.WriteString(`<p>You, and:</p><ul class="sw-plain sw-rows">`)
	for _, p := range people {
		name, _ := p.Fields["name"].(string)
		access, _ := p.Fields["access"].(string)
		b.WriteString(`<li class="sw-cluster"><span><strong>` + template.HTMLEscapeString(name) + `</strong> ` + accessWords(access) +
			`. <span class="sw-muted sw-small">` + template.HTMLEscapeString(s.howTheyComeIn(p)) + `</span></span>` +
			string(s.form(ui.Form{Action: "/access/take", Hidden: ui.Hidden("id", p.ID), Button: &ui.Button{Label: "Take away access", Context: name, Variant: ui.Quiet}})) + `</li>`)
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}

// accessWords is what someone let in may do, in a few words.
func accessWords(access string) string {
	switch access {
	case records.View:
		return "may look"
	case records.Host:
		return "may edit, and keep a copy on their own computer"
	}
	return "may edit"
}

// peopleLetIn is everyone with access to this workspace, by name.
func (s *Server) peopleLetIn() []*store.Record {
	if _, ok := s.app.Store.Types().Get(records.PersonType); !ok {
		return nil
	}
	all, _ := s.app.Store.List(records.PersonType, store.ListOptions{})
	var out []*store.Record
	for _, p := range all {
		if a, _ := p.Fields["access"].(string); a == records.View || a == records.Edit || a == records.Host {
			out = append(out, p)
		}
	}
	return out
}

// howTheyComeIn says the ways a person reaches the workspace: Tailscale,
// their devices on the Wi-Fi, an invite not opened yet.
func (s *Server) howTheyComeIn(p *store.Record) string {
	var ways []string
	if email, _ := p.Fields["email"].(string); email != "" {
		ways = append(ways, "On Tailscale as "+email)
	}
	devices := 0
	for _, d := range s.lanDevices() {
		if d.Person == p.ID {
			devices++
		}
	}
	switch {
	case devices == 1:
		ways = append(ways, "1 device on the Wi-Fi")
	case devices > 1:
		ways = append(ways, fmt.Sprintf("%d devices on the Wi-Fi", devices))
	}
	for _, i := range s.lanInvites() {
		if i.Person == p.ID {
			ways = append(ways, "An invite not opened yet, until "+i.Until.Format("2 Jan"))
		}
	}
	if len(ways) == 0 {
		return "Not come in yet."
	}
	return strings.Join(ways, ". ") + "."
}

// accessTake is Take away access: the person's devices and Tailscale
// login stop opening the workspace at once, and an invite not yet opened
// no longer works. Undo gives the access back; the invite stays gone.
func (s *Server) accessTake(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.PostForm.Get("id")
	name, entry, err := s.app.Chat.TakeAccess(id)
	if err != nil {
		s.failed(w, r, "Access not changed", err, "/workspaces#ws-access")
		return
	}
	var kept []lanInvite
	for _, i := range s.lanInvites() {
		if i.Person != id {
			kept = append(kept, i)
		}
	}
	s.saveLanInvites(kept)
	s.tell(w, r, outcome{Title: "Access taken away", Text: name + " can no longer open this workspace, from any device.", Undo: entry}, "/workspaces#ws-access")
}
