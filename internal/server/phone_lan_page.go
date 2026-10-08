package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"rsc.io/qr"
)

// phoneSection is Use Sameway on your phone on Workspaces: the switch, and
// once on, the code to scan and the phones paired.
func (s *Server) phoneSection(r *http.Request) string {
	if s.fleet == nil || s.fleet.LAN == nil || !s.chatFor(r).IsOwner() {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-phone"><h2 id="ws-phone">On your phone</h2>`)
	base := s.fleet.LANBase()
	if base == "" {
		b.WriteString(`<p>Use Sameway on a phone or tablet on the same Wi-Fi as this computer. Nothing on the Wi-Fi reaches your workspace unless you pair it here; Windows may ask whether Sameway may use the network: choose Allow.</p>`)
		b.WriteString(`<form method="post" action="/phone/on">` + string(s.component("button", map[string]any{"label": "Use Sameway on your phone", "type": "submit", "variant": "secondary"})) + `</form></section>`)
		return b.String()
	}
	link := base + "/pair?code=" + lanPairCode()
	b.WriteString(`<p>Scan this with your phone's camera, on the same Wi-Fi as this computer. It works once, for ten minutes; reload this page for another.</p>`)
	b.WriteString(`<figure class="sw-stack">` + qrSVG(link) + `<figcaption class="sw-small sw-muted">Or open ` + template.HTMLEscapeString(link) + ` on the phone.</figcaption></figure>`)
	if devices := s.lanDevices(); len(devices) > 0 {
		b.WriteString(`<h3>Paired</h3><ul class="sw-plain sw-rows">`)
		for _, d := range devices {
			b.WriteString(`<li class="sw-cluster">` + template.HTMLEscapeString(d.Name) + ` <span class="sw-muted sw-small">since ` + d.Added.Format("2 Jan") + `</span><form method="post" action="/phone/forget"><input type="hidden" name="id" value="` + d.ID + `">` +
				string(s.component("button", map[string]any{"label": "Remove", "context": d.Name + " paired " + d.Added.Format("2 Jan 15:04"), "type": "submit", "variant": "quiet"})) + `</form></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<form method="post" action="/phone/off">` + string(s.component("button", map[string]any{"label": "Stop answering on the Wi-Fi", "type": "submit", "variant": "secondary"})) + `</form></section>`)
	return b.String()
}

// qrSVG draws a QR code as an image a screen reader names.
func qrSVG(text string) string {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return ""
	}
	n := c.Size
	var b strings.Builder
	fmt.Fprintf(&b, `<svg role="img" aria-label="Code to scan with your phone" viewBox="-4 -4 %d %d" width="220" height="220" style="background:#fff" shape-rendering="crispEdges"><path fill="#000" d="`, n+8, n+8)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if c.Black(x, y) {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x, y)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String()
}

var errNoLAN = errors.New("this Sameway was not started in a way that can answer on the Wi-Fi")

// phoneOn, phoneOff and phoneForget are the section's forms.
func (s *Server) phoneOn(w http.ResponseWriter, r *http.Request) {
	if s.fleet == nil || s.fleet.LAN == nil {
		s.failed(w, r, "Not changed", errNoLAN, "/workspaces")
		return
	}
	if err := s.fleet.LAN(true); err != nil {
		s.failed(w, r, "Not changed", err, "/workspaces")
		return
	}
	s.setSetting("server.lan", "on")
	s.tell(w, r, outcome{Title: "Sameway answers on the Wi-Fi", Text: "Scan the code below with your phone. Nothing reaches your workspace unless it is paired."}, "/workspaces#ws-phone")
}

func (s *Server) phoneOff(w http.ResponseWriter, r *http.Request) {
	if s.fleet != nil && s.fleet.LAN != nil {
		s.fleet.LAN(false)
	}
	s.setSetting("server.lan", "off")
	s.tell(w, r, outcome{Title: "Sameway answers on this computer only", Text: "Phones paired stay paired for when you turn it on again; remove them here."}, "/workspaces")
}

func (s *Server) phoneForget(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.PostForm.Get("id")
	var kept []lanDevice
	gone := ""
	for _, d := range s.lanDevices() {
		if d.ID == id {
			gone = d.Name
			continue
		}
		kept = append(kept, d)
	}
	s.saveLanDevices(kept)
	if gone != "" {
		chat.Record(s.app.Store, "human", chat.Change{Action: "removed", Component: "device", Detail: gone})
	}
	s.tell(w, r, outcome{Title: "Removed", Text: gone + " no longer opens Sameway."}, "/workspaces#ws-phone")
}

func (s *Server) setSetting(key, value string) {
	if s.app.Chat.SetSetting != nil {
		s.app.Chat.SetSetting(key, value)
	}
}

func (s *Server) phoneRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /phone/on", s.phoneOn)
	m.HandleFunc("POST /phone/off", s.phoneOff)
	m.HandleFunc("POST /phone/forget", s.phoneForget)
}
