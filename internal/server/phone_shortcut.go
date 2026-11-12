package server

import (
	"errors"
	"html/template"
	"net"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// An iPhone lets no web app into its Share menu, but it lets in a
// Shortcut, and a Shortcut can send what is shared to Save to Sameway.
// It has no cookie of a phone paired, so it carries a key of its own in
// its link: one that lets in nothing but a share (phone_lan.go), listed
// with the phones paired on Workspaces, where it can be removed.

// shortcutOnly marks a key that lets in a share and nothing else.
const shortcutOnly = "share"

// shortcutOffer is the way to make one, for the owner, on Save to Sameway.
func (s *Server) shortcutOffer(r *http.Request) string {
	if !s.chatFor(r).IsOwner() {
		return ""
	}
	if fromThisComputer(r) {
		// The link it makes is pasted on the iPhone, so it is made there.
		return `<p>Make it on the iPhone, where its link is pasted: open this page in Sameway there (<a class="sw-link" href="/workspaces#ws-phone">On your phone</a> says how) and press Make a Shortcut.</p>`
	}
	return string(s.form(ui.Form{Action: "/phone/shortcut", Button: &ui.Button{Label: "Make a Shortcut for an iPhone's Share menu", Variant: ui.Secondary}}))
}

// shortcutBase is the address a phone reaches this computer by: the one
// asked from, unless that is this computer itself, then the Wi-Fi's.
func (s *Server) shortcutBase(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return s.origin(r)
	}
	if s.fleet != nil {
		return s.fleet.LANBase()
	}
	return ""
}

// phoneShortcut makes the key, and shows its link once, with how to make
// the Shortcut on the iPhone.
func (s *Server) phoneShortcut(w http.ResponseWriter, r *http.Request) {
	base := s.shortcutBase(r)
	if base == "" {
		s.failed(w, r, "No Shortcut made", errors.New("a phone cannot reach this computer yet: under Workspaces, choose Use Sameway on your phone first"), "/share")
		return
	}
	secret := token(32)
	s.saveLanDevices(append(s.lanDevices(), lanDevice{ID: token(6), Hash: hashOf(secret), Name: "iPhone Shortcut", Added: s.now(), Only: shortcutOnly}))
	records.Record(s.app.Store, "human", records.Change{Action: "paired", Component: "device", Detail: "iPhone Shortcut"})
	link := base + "/share?key=" + secret
	if pageAction(r) {
		tellJSON(w, outcome{Title: "Shortcut link made", Text: "In the iPhone's Shortcuts app, make a Shortcut shown in the Share Sheet whose one action is Get Contents of URL: this link, " + link + ", Method POST, Request Body Form, with a File field files set to Shortcut Input."}, "/share")
		return
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<p>Do this on the iPhone, in the Shortcuts app. The link below is shown only now: copy it first. It lets in photos and files shared, nothing else, and can be removed under Workspaces.</p>`)
	b.WriteString(string(s.part(ui.TextField{Label: "The Shortcut's link", Name: "link", Value: link, Spellcheck: ui.Bool(false)})))
	b.WriteString(`<ol class="sw-stack">` +
		`<li>In Shortcuts, press + to make a new Shortcut, and name it Sameway.</li>` +
		`<li>Press the Shortcut's details (the i or the arrow by its name), turn on Show in Share Sheet, and let it receive Images, Media and Files.</li>` +
		`<li>Add the action Get Contents of URL, and paste the link as its URL.</li>` +
		`<li>Press Show More: set Method to POST and Request Body to Form. Add a field of the kind File, with the key <strong>files</strong> and the value Shortcut Input.</li>` +
		`<li>Press Done. Now in Photos, or any app, Share then Sameway saves what you chose here.</li>` +
		`</ol>`)
	b.WriteString(`<p class="sw-muted">The link is for ` + esc(base) + `: on the home Wi-Fi, it works while the iPhone is on the same Wi-Fi as this computer.</p>`)
	s.page(w, r, "A Shortcut for the Share menu", template.HTML(b.String()), pageOptions{Lede: "Share photos from an iPhone straight to Sameway."})
}
