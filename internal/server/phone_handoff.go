package server

import (
	"errors"
	"html/template"
	"net"
	"net/http"
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A thing found on the phone is better read or worked on at the
// computer's keyboard, and a thing open on the computer is wanted on the
// way out of the door. So a record's page on a phone has Open on the
// computer, which opens the same page on the computer's own screen, and
// on the computer has On your phone, a code to scan that opens it on the
// phone. Each shows only where it can work: the first on another device,
// when Sameway was opened on the computer (not served headless); the
// second on the computer, while the Wi-Fi answers a paired phone.

// fromThisComputer says the request came from the computer Sameway runs on.
func fromThisComputer(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// handoff is the press that takes a page to the other device, for the
// owner, where it can.
func (s *Server) handoff(r *http.Request, path string) string {
	if s.fleet == nil || !s.chatFor(r).IsOwner() {
		return ""
	}
	if !fromThisComputer(r) && s.fleet.Show != nil {
		return string(s.form(ui.Form{Action: "/computer/show", Hidden: ui.Hidden("to", path), Button: &ui.Button{Label: "Open on the computer", Variant: ui.Quiet}}))
	}
	if fromThisComputer(r) && s.fleet.LANBase != nil && s.fleet.LANBase() != "" {
		return string(s.part(ui.Link{Href: "/phone/here?to=" + url.QueryEscape(path), Label: "On your phone"}))
	}
	return ""
}

// computerShow opens a page of this workspace on the computer's screen.
func (s *Server) computerShow(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	to := r.PostForm.Get("to")
	if !web.Local(to) {
		s.failed(w, r, "Not opened", errors.New("only a page of this workspace opens on the computer"), "/")
		return
	}
	if s.fleet == nil || s.fleet.Show == nil {
		s.failed(w, r, "Not opened", errors.New("this Sameway runs without a screen of its own: open it on the computer with sameway open"), to)
		return
	}
	if err := s.fleet.Show(to); err != nil {
		s.failed(w, r, "Not opened", err, to)
		return
	}
	s.tell(w, r, outcome{Title: "Open on the computer", Text: "It is on the computer's screen now."}, to)
}

// phoneHere is a code to scan that opens a page on a paired phone.
func (s *Server) phoneHere(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to")
	if !web.Local(to) {
		to = "/"
	}
	base := ""
	if s.fleet != nil && s.fleet.LANBase != nil {
		base = s.fleet.LANBase()
	}
	if base == "" {
		s.failed(w, r, "Not on the phone", errNoLAN, to)
		return
	}
	link := base + to
	body := `<p>Scan this with your phone's camera, on the same Wi-Fi: the page opens there. A phone not paired yet is asked to pair first.</p>` +
		`<figure class="sw-stack">` + qrSVG(link) + `<figcaption><code>` + template.HTMLEscapeString(link) + `</code></figcaption></figure>` +
		`<p><a class="sw-link" href="` + template.HTMLEscapeString(to) + `">Back</a></p>`
	s.page(w, r, "On your phone", template.HTML(body), pageOptions{})
}
