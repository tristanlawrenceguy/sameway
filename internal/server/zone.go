package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A workspace's days are its person's, in the time zone they plan by
// (workspace.Zone), not the zone of the computer it runs on, which may be
// a server elsewhere or a laptop that travels. The owner chooses it on
// Help; the assistant and an agent set it with set_setting ui.zone.
// Unset, it is this computer's (design/foundations/today.md).

// zoneWords is a zone as a person says it: Madrid time, New York time.
func zoneWords(name string) string {
	switch name {
	case "", "Local":
		return "this computer's time"
	case "UTC":
		return "UTC"
	}
	return strings.ReplaceAll(name[strings.LastIndex(name, "/")+1:], "_", " ") + " time"
}

// zoneLine is the time zone's line on Help, for the owner.
func (s *Server) zoneLine() string {
	name := s.app.Workspace.Config.UI.Zone
	said := "Time zone: " + zoneWords(name)
	if name != "" && name != "Local" && name != "UTC" {
		said += " (" + name + ")"
	}
	said += ". Today, what is late, tomorrow, repeats, the morning brief and the weekly review follow it."
	return template.HTMLEscapeString(said) + " " + string(s.form(ui.Form{Action: "/zone", Class: "sw-stack",
		Body:   s.part(ui.TextField{Label: "Time zone", Name: "zone", ID: "zone-name", Value: name, Hint: "A place's name, such as Europe/Madrid or America/New_York. Empty follows this computer."}),
		Button: &ui.Button{Label: "Set the time zone", Variant: ui.Secondary}}))
}

// zoneSet changes the workspace's time zone, logged so it can be undone.
// It applies to the next thing worked out: no restart.
func (s *Server) zoneSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	zone := strings.TrimSpace(r.PostForm.Get("zone"))
	if zone != "" && !workspace.KnownZone(zone) {
		s.failed(w, r, "Not set", errors.New("give the time zone by a place's name, such as Europe/Madrid or America/New_York"), "/help")
		return
	}
	undo, _, err := s.apply(r, records.Change{Action: "set", Component: "ui.zone", Detail: zone}, records.Op{Type: records.SettingOp, ID: "ui.zone", After: map[string]any{"value": zone}})
	if err != nil {
		s.failed(w, r, "Not set", err, "/help")
		return
	}
	s.tell(w, r, outcome{Title: "Times are now " + zoneWords(zone), Text: "Today, the morning brief and the weekly review follow it.", Undo: undo}, "/help")
}
