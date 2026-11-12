package server

import (
	"context"
	"errors"
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// The home's devices from a phone, with no maker's cloud between. While
// no hub is set, the running program looks for one on the home Wi-Fi
// every few minutes (devices.Find, given by cli/background.go); when it finds one, Workspaces offers it in one
// press, saved with the subscriptions that take in Zigbee2MQTT's and
// Tasmota's devices, and connected at once (app/devices.go). With
// neither a hub set nor one found, Workspaces says nothing of it. A
// device that says it is on or off has a button on its page to turn it
// the other way.

// hubsFound are the brokers last found on the home network, and how they
// are looked for: nil looks for none.
type hubsFound struct {
	mu   sync.Mutex
	find func(ctx context.Context) ([]devices.Broker, error)
	list []devices.Broker
}

// LookForHubs is how a hub is looked for on the home network: the real
// program gives devices.Find (cli/background.go); a test its own. Until
// it is given, nothing listens on the network.
func (s *Server) LookForHubs(find func(ctx context.Context) ([]devices.Broker, error)) {
	s.hubs.mu.Lock()
	s.hubs.find = find
	s.hubs.mu.Unlock()
}

// lookForHub looks for a broker on the home network, while none is set.
func (s *Server) lookForHub(ctx context.Context) error {
	s.hubs.mu.Lock()
	find := s.hubs.find
	s.hubs.mu.Unlock()
	if find == nil || !s.hasDevices() || strings.TrimSpace(s.app.Workspace.Config.MQTT.Broker) != "" {
		return nil
	}
	found, err := find(ctx)
	s.hubs.mu.Lock()
	s.hubs.list = found
	s.hubs.mu.Unlock()
	return err
}

// hasDevices says whether the workspace keeps devices.
func (s *Server) hasDevices() bool {
	_, ok := s.app.Types.Get(devices.DeviceType)
	return ok
}

// devicesSection is Your home's devices on Workspaces: the hub in use, or
// the one found, ready to press; nothing when there is neither.
func (s *Server) devicesSection(r *http.Request) string {
	if !s.chatFor(r).IsOwner() || !s.hasDevices() {
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-devices"><h2 id="ws-devices">Your home's devices</h2>`)
	if broker := s.app.Workspace.Config.MQTT.Broker; strings.TrimSpace(broker) != "" {
		b.WriteString(`<p>Your lights, plugs and sensors come in through your hub at ` + esc(plainHub(broker)) + `. ` + string(s.part(ui.Link{Href: "/t/" + devices.DeviceType, Label: "See your devices"})) + `</p>`)
		if live, err := s.app.DevicesLive(); err != nil {
			b.WriteString(`<p>It is not connected now: ` + esc(err.Error()) + `</p>`)
		} else if !live {
			b.WriteString(`<p>It connects the next time Sameway starts.</p>`)
		}
		b.WriteString(string(s.form(ui.Form{Action: "/devices/hub", Hidden: ui.Hidden("broker", ""), Button: &ui.Button{Label: "Stop using this hub", Variant: ui.Secondary}})) + `</section>`)
		return b.String()
	}
	s.hubs.mu.Lock()
	found := s.hubs.list
	s.hubs.mu.Unlock()
	if len(found) == 0 {
		return ""
	}
	hub := found[0]
	b.WriteString(`<p>Sameway found a hub for your home's devices on your Wi-Fi: ` + esc(hub.Name) + ` at ` + esc(plainHub(hub.Addr)) + `. Using it puts your lights, plugs and sensors on pages of their own, and the ones that switch get a button to turn them on and off.</p>`)
	b.WriteString(string(s.form(ui.Form{Action: "/devices/hub", Hidden: ui.Hidden("broker", hub.Addr), Button: &ui.Button{Label: "Use this hub", Variant: ui.Secondary}})) + `</section>`)
	return b.String()
}

// plainHub is a broker's address as a person reads it: 192.168.1.10:1883.
func plainHub(broker string) string {
	if _, rest, ok := strings.Cut(broker, "://"); ok {
		return rest
	}
	return broker
}

// hubAddress is a broker as the setting takes it, from an address a
// person or a page gives: tcp:// and the usual port when left out.
func hubAddress(given string) string {
	given = strings.TrimSpace(given)
	if given == "" {
		return ""
	}
	if !strings.Contains(given, "://") {
		given = "tcp://" + given
	}
	if _, _, err := net.SplitHostPort(plainHub(given)); err != nil {
		given += ":1883"
	}
	return given
}

// devicesHub uses a hub, or stops using it: the broker saved, with the
// common subscriptions when there are none yet, and connected at once
// while Sameway runs.
func (s *Server) devicesHub(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	broker := hubAddress(r.PostForm.Get("broker"))
	const back = "/workspaces#ws-devices"
	if s.app.Records.SetSetting == nil {
		s.failed(w, r, "Not changed", errors.New("this workspace has no settings file"), back)
		return
	}
	if broker != "" && len(s.app.Workspace.Config.MQTT.Subscribe) == 0 {
		if err := s.app.Records.SetSetting("mqtt.subscribe", strings.Join(devices.Common, ", ")); err != nil {
			s.failed(w, r, "Not changed", err, back)
			return
		}
	}
	if err := s.app.Records.SetSetting("mqtt.broker", broker); err != nil {
		s.failed(w, r, "Not changed", err, back)
		return
	}
	if broker == "" {
		s.tell(w, r, outcome{Title: "Your hub is not used", Text: "Your devices stay in your lists as they last said they were."}, back)
		return
	}
	switch live, err := s.app.DevicesLive(); {
	case err != nil:
		s.failed(w, r, "Saved, but not connected", err, back)
	case !live:
		s.tell(w, r, outcome{Title: "Your hub is saved", Text: "It connects the next time Sameway starts."}, back)
	default:
		s.tell(w, r, outcome{Title: "Your hub is connected", Text: "Each device comes into your devices the next time it says something."}, back)
	}
}

// deviceSwitch is the press on a device's page that turns it on or off,
// for the owner, when it says which it is and Sameway knows how.
func (s *Server) deviceSwitch(r *http.Request, rec *store.Record) string {
	if !s.chatFor(r).IsOwner() {
		return ""
	}
	on, said := devices.OnOff(rec.Fields)
	if _, _, ok := devices.Command(rec.Fields, !on); !said || !ok {
		return ""
	}
	name, _ := rec.Fields["name"].(string)
	now, label, to := "It is off.", "Turn on", "on"
	if on {
		now, label, to = "It is on.", "Turn off", "off"
	}
	return string(s.form(ui.Form{Action: "/devices/" + rec.ID + "/switch", Class: "sw-stack", Hidden: ui.Hidden("to", to),
		Body: template.HTML(`<p>` + now + `</p>`), Button: &ui.Button{Label: label, Context: name}}))
}

// deviceSwitched turns a device on or off from its page.
func (s *Server) deviceSwitched(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, to := r.PathValue("id"), r.PostForm.Get("to")
	back := "/t/" + devices.DeviceType + "/" + id
	if to != "on" && to != "off" {
		s.failed(w, r, "Not switched", errors.New("say on or off"), back)
		return
	}
	said, err := s.app.Automate.SwitchDevice(id, to == "on")
	if err != nil {
		s.failed(w, r, "Not switched", err, back)
		return
	}
	s.tell(w, r, outcome{Title: said, Text: "Its state here changes when it answers."}, back)
}
