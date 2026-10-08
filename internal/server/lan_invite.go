package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Someone else in the house, on the same Wi-Fi: the owner names them and
// says whether they may look or edit, and Sameway makes a link that works
// once, for a week, to send them. Opened, it leaves their device a cookie
// that lets them in as that person, with what their person record says
// they may do; taking the access away, or removing the device, shuts it.
// Tailscale is no longer needed for a family or a small office.

// lanInviteFor is how long an invite works: long enough to be sent and
// opened on another day, once.
const lanInviteFor = 7 * 24 * time.Hour

type lanInvite struct {
	Hash   string    `json:"hash"`
	Person string    `json:"person"`
	Until  time.Time `json:"until"`
}

func (s *Server) lanInvites() []lanInvite {
	var out []lanInvite
	json.Unmarshal([]byte(s.app.Store.Meta("lan:invites")), &out)
	return out
}

func (s *Server) saveLanInvites(in []lanInvite) {
	var kept []lanInvite
	for _, i := range in {
		if time.Now().Before(i.Until) {
			kept = append(kept, i)
		}
	}
	b, _ := json.Marshal(kept)
	s.app.Store.SetMeta("lan:invites", string(b))
}

// inviteForm is Invite someone, under the phone's code on Workspaces.
func (s *Server) inviteForm() string {
	return `<h3>Someone else on this Wi-Fi</h3><p>Invite someone in your home or office to open this workspace from their own phone or computer. They see only what you let them: to look, or to edit. Your conversations with the assistant stay yours.</p>` +
		`<form method="post" action="/phone/invite" class="sw-stack">` +
		string(s.component("text-field", map[string]any{"label": "Their name", "name": "name", "required": true, "autocomplete": "off"})) +
		string(s.component("select", map[string]any{"label": "They may", "name": "access", "value": chat.Edit, "as": "radios",
			"options": []any{map[string]any{"value": chat.Edit, "label": "Edit: add and change things"}, map[string]any{"value": chat.View, "label": "Look: read only"}}})) +
		string(s.component("button", map[string]any{"label": "Make an invite", "type": "submit", "variant": "secondary"})) + `</form>`
}

// phoneInvite makes the person's access and the link that lets them in.
func (s *Server) phoneInvite(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	base := ""
	if s.fleet != nil {
		base = s.fleet.LANBase()
	}
	if base == "" {
		s.failed(w, r, "No invite made", errNoLAN, "/workspaces#ws-phone")
		return
	}
	p, err := s.app.Chat.GiveAccess(r.PostForm.Get("name"), r.PostForm.Get("access"))
	if err != nil {
		s.failed(w, r, "No invite made", err, "/workspaces#ws-phone")
		return
	}
	name, _ := p.Fields["name"].(string)
	code := token(16)
	s.saveLanInvites(append(s.lanInvites(), lanInvite{Hash: hashOf(code), Person: p.ID, Until: time.Now().Add(lanInviteFor)}))
	link := base + "/pair?invite=" + code
	said := "Send " + name + " this link: " + link + ". It works once, for a week, on this Wi-Fi."
	if pageAction(r) {
		tellJSON(w, outcome{Title: "Invite made", Text: said}, "/workspaces#ws-phone")
		return
	}
	esc := template.HTMLEscapeString
	body := `<p>Send ` + esc(name) + ` this link by message or email, or let them scan the code. It works once, within a week, on a device on the same Wi-Fi as this computer.</p>` +
		`<p><code>` + esc(link) + `</code></p><figure class="sw-stack">` + qrSVG(link) + `</figure>` +
		`<p>To shut them out, remove their device under Workspaces. To change what they may do, make them another invite. <a class="sw-link" href="/workspaces#ws-phone">Back to Workspaces</a></p>`
	s.page(w, r, "Invite for "+name, template.HTML(body), pageOptions{})
}

// lanJoin is an invite opened: the device is let in as the person.
func (s *Server) lanJoin(w http.ResponseWriter, r *http.Request) {
	h := hashOf(r.URL.Query().Get("invite"))
	var kept []lanInvite
	var found *lanInvite
	for _, i := range s.lanInvites() {
		if i.Hash == h && found == nil {
			i := i
			found = &i
			continue
		}
		kept = append(kept, i)
	}
	if found == nil || time.Now().After(found.Until) {
		lanSay(w, "This invite has been used or is more than a week old. Ask for a new one.")
		return
	}
	s.saveLanInvites(kept)
	p, err := s.app.Store.Get(chat.PersonType, found.Person)
	if err != nil {
		lanSay(w, "The one this invite was for is no longer in this workspace.")
		return
	}
	name, _ := p.Fields["name"].(string)
	secret := token(32)
	device := name + "'s " + phoneName(r.UserAgent())
	s.saveLanDevices(append(s.lanDevices(), lanDevice{ID: token(6), Hash: hashOf(secret), Name: device, Person: p.ID, Added: time.Now()}))
	chat.Record(s.app.Store, "human", chat.Change{Action: "paired", Component: "device", Detail: device})
	http.SetCookie(w, &http.Cookie{Name: lanCookie, Value: secret, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 400 * 24 * 3600})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// lanVisitor is who a paired device lets in: the owner, or the person it
// was invited for, with what they may do now.
func (s *Server) lanVisitor(d lanDevice) (chat.Visitor, bool) {
	if d.Person == "" {
		return chat.Visitor{Access: chat.Owner, Name: "Owner", Device: d.Name}, true
	}
	p, err := s.app.Store.Get(chat.PersonType, d.Person)
	if err != nil {
		return chat.Visitor{}, false
	}
	access, _ := p.Fields["access"].(string)
	if access != chat.View && access != chat.Edit {
		return chat.Visitor{}, false
	}
	name, _ := p.Fields["name"].(string)
	return chat.Visitor{Person: p.ID, Name: name, Login: "device:" + d.ID, Access: access, Device: d.Name}, true
}

// mayWords is what an invited device may do, said beside it.
func (s *Server) mayWords(d lanDevice) string {
	if d.Person == "" {
		return ""
	}
	v, ok := s.lanVisitor(d)
	switch {
	case !ok:
		return ", shut out"
	case v.Access == chat.View:
		return ", may look"
	}
	return ", may edit"
}

// lanSay is a page for someone not let in.
func lanSay(w http.ResponseWriter, words string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`<!doctype html><meta name="viewport" content="width=device-width"><title>Sameway</title><p style="font:1.1rem system-ui;max-width:30rem;margin:2rem auto;padding:0 1rem">` + template.HTMLEscapeString(words) + `</p>`))
}
