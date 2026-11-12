package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Sameway on a phone's home screen, from the Wi-Fi. A phone just paired
// is shown, once, how to put it there, in the words of that phone. Two
// things stood in the way. The browser asks for the manifest (the app's
// name, icon and window) without the phone's cookie unless told to, so
// on the Wi-Fi it got the pairing page and the home screen got a plain
// bookmark: the page now asks with the cookie (layout.html), and the
// icons answer anyone, holding nothing of the workspace. And an iPhone
// keeps a home-screen app's cookies apart from Safari's, so the app
// opened unpaired: the manifest a paired phone gets starts the app at a
// link that pairs it, once, as a device of its own.

// appParts are what an app is made of besides the manifest.
var appParts = map[string]bool{"/favicon.svg": true, "/favicon.ico": true, "/icon-192.png": true, "/icon-512.png": true, "/icon-square-180.png": true, "/icon-square-512.png": true}

// deviceOf is the paired phone a request comes from, by its cookie.
func (s *Server) deviceOf(r *http.Request) (lanDevice, bool) {
	c, err := r.Cookie(lanCookie)
	if err != nil {
		return lanDevice{}, false
	}
	h := hashOf(c.Value)
	for _, d := range s.lanDevices() {
		if d.Hash == h && d.Only == "" {
			return d, true
		}
	}
	return lanDevice{}, false
}

// lanManifest is the manifest on the Wi-Fi: for a paired phone, starting
// at the link that pairs its home-screen app; for anything else, plain.
func (s *Server) lanManifest(w http.ResponseWriter, r *http.Request) {
	d, ok := s.deviceOf(r)
	if !ok {
		appManifest(w, r)
		return
	}
	if d.App == "" {
		d.App = token(24)
		devices := s.lanDevices()
		for i := range devices {
			if devices[i].ID == d.ID {
				devices[i].App = d.App
			}
		}
		s.saveLanDevices(devices)
	}
	more := ""
	if d.Person == "" {
		more = `,{"name":"Send to the computer","short_name":"To computer","description":"Open a link or paste words on the computer","url":"/computer/send"},` +
			`{"name":"Scan to the list","short_name":"Scan","description":"Take a photo of something you need and it goes on the shopping list","url":"/scan"}`
	}
	v, _ := s.lanVisitor(d)
	manifestStarting(w, "/pair?app="+d.App, more, v.Access == records.View)
}

// lanApp is the home-screen app opening: paired already, it goes in; the
// first time, the link pairs it as a device of its own, for the same
// person, and works no more.
func (s *Server) lanApp(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.deviceOf(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	app := r.URL.Query().Get("app")
	devices := s.lanDevices()
	for i, d := range devices {
		if app == "" || d.App != app {
			continue
		}
		devices[i].App = ""
		secret := token(32)
		devices = append(devices, lanDevice{ID: token(6), Hash: hashOf(secret), Name: d.Name + " app", Added: s.now(), Person: d.Person})
		s.saveLanDevices(devices)
		http.SetCookie(w, &http.Cookie{Name: lanCookie, Value: secret, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 400 * 24 * 3600})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	lanPairPage(w, "This app is not paired yet. On the computer, open Workspaces in Sameway, choose Use Sameway on your phone, and type the letters it shows here.")
}

// installPage is how to put Sameway on this phone's home screen, in its
// own words, shown once after pairing; on a computer, how to install it.
func (s *Server) installPage(w http.ResponseWriter, r *http.Request) {
	ua := strings.ToLower(r.UserAgent())
	var how string
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		how = `<ol class="sw-stack"><li>Press Share, the square with an arrow: at the bottom of Safari, or under the three dots beside the address (in Chrome, at the top right).</li><li>Choose Add to Home Screen; scroll down the list if it is not in sight.</li><li>Press Add. Sameway is on your home screen, and opens without the browser around it.</li></ol>`
	case strings.Contains(ua, "android"):
		how = `<p><button type="button" class="sw-button sw-button--primary sw-pressable" data-install hidden>Add Sameway to your home screen</button></p>` +
			`<ol class="sw-stack"><li>Press the browser's menu, the three dots at the top right.</li><li>Choose Add to Home screen, or Install app.</li><li>Press Add. Sameway is on your home screen.</li></ol>`
	default:
		how = `<p><button type="button" class="sw-button sw-button--primary sw-pressable" data-install hidden>Install Sameway as an app</button></p>` +
			`<p>In Chrome or Edge, the install icon at the end of the address bar does the same.</p>`
	}
	body := `<p>Put Sameway on your home screen, so it is one press away, like your other apps.</p>` + how +
		`<p><a class="sw-link" href="/">Not now: open Sameway</a></p>`
	s.page(w, r, "On your home screen", template.HTML(body), pageOptions{})
}
