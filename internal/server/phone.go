package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/notify"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A reminder reached the person only at this computer; sending it to a
// phone meant writing a command line by hand (notify.command). Now Help
// offers Send reminders to my phone: one press makes a long random ntfy
// topic, sets notify.phone, sends a first message to it, and says how to
// subscribe, in the free ntfy app or in a browser. The words of each
// reminder go through ntfy.sh, which the page says before the press.

// phoneLine is the phone's line on Help, for the owner.
func (s *Server) phoneLine() string {
	esc := template.HTMLEscapeString
	if topic := s.app.Workspace.Config.Notify.Phone; topic != "" {
		return `Your phone: reminders are also sent to <a class="sw-link" href="` + esc(topic) + `">` + esc(topic) + `</a>; subscribe to it in the free ntfy app to have them there. ` +
			string(s.form(ui.Form{Action: "/notify/phone", Hidden: ui.Hidden("set", "off"), Button: &ui.Button{Label: "Stop sending reminders to my phone", Variant: ui.Secondary}}))
	}
	return `Your phone: reminders ring only on this computer. They can also go to your phone through ntfy, a free notification service with an app for every phone; the words of each reminder then pass through ntfy.sh. ` +
		string(s.form(ui.Form{Action: "/notify/phone", Hidden: ui.Hidden("set", "on"), Button: &ui.Button{Label: "Send reminders to my phone", Variant: ui.Secondary}}))
}

// phoneSet makes a topic and sends the first message to it, or stops.
func (s *Server) phoneSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if s.app.Records.SetSetting == nil {
		s.failed(w, r, "Not changed", errors.New("this workspace has no settings file"), "/help")
		return
	}
	if r.PostForm.Get("set") != "on" {
		if err := s.app.Records.SetSetting("notify.phone", ""); err != nil {
			s.failed(w, r, "Not changed", err, "/help")
			return
		}
		s.tell(w, r, outcome{Title: "Reminders stay on this computer", Text: "Nothing more is sent to your phone; you can unsubscribe in the ntfy app."}, "/help")
		return
	}
	b := make([]byte, 12)
	rand.Read(b)
	topic := notify.NtfyServer + "/sameway-" + hex.EncodeToString(b)
	if err := notify.Phone(topic, "Sameway", "Reminders from "+s.app.Workspace.Config.Name+" arrive here."); err != nil {
		s.failed(w, r, "Not set up", errors.New("ntfy could not be reached ("+err.Error()+"); try again when this computer is online"), "/help")
		return
	}
	if err := s.app.Records.SetSetting("notify.phone", topic); err != nil {
		s.failed(w, r, "Not set up", err, "/help")
		return
	}
	s.tell(w, r, outcome{Title: "Reminders go to your phone too",
		Text: "Install the free ntfy app on your phone and subscribe to " + topic + ", or open that address in your phone's browser. A first message is waiting there."}, "/help")
}
