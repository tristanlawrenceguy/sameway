package server

import (
	"net/http"
	"os"

	"github.com/tristanlawrenceguy/sameway/internal/atlogin"
)

// Reminders ring and scheduled actions run only while Sameway runs. The
// owner can have it open, without a browser tab, whenever they sign in to
// this computer (internal/atlogin); the switch is on Workspaces, beside
// Quit Sameway, and what a reminder made says it when it is off.

// atLoginSection is the switch, for a system that has a place for it.
func (s *Server) atLoginSection() string {
	if atlogin.Path() == "" {
		return ""
	}
	label, value, said := "Open Sameway when I sign in", "on", "Sameway opens only when you open it, so reminders ring only then."
	if atlogin.On() {
		label, value, said = "Stop opening Sameway when I sign in", "off", "Sameway opens when you sign in to this computer, without a browser tab, so reminders ring all day."
	}
	return `<form method="post" action="/at-login" class="sw-stack"><input type="hidden" name="set" value="` + value + `">` +
		string(s.component("button", map[string]any{"label": label, "type": "submit", "variant": "secondary"})) +
		`<p class="sw-small sw-muted">` + said + `</p></form>`
}

// atLoginSet turns opening at sign-in on or off for this workspace.
func (s *Server) atLoginSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	on := r.PostForm.Get("set") == "on"
	exe, err := os.Executable()
	if err == nil {
		err = atlogin.Set(on, exe, s.app.Workspace.Dir)
	}
	if err != nil {
		s.failed(w, r, "Not changed", err, "/workspaces")
		return
	}
	o := outcome{Title: "Sameway opens when you sign in", Text: "It opens this workspace without a browser tab, so reminders ring all day. Workspaces turns it off."}
	if !on {
		o = outcome{Title: "Sameway no longer opens when you sign in", Text: "Reminders ring while you have it open."}
	}
	s.tell(w, r, o, "/workspaces")
}
