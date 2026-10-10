package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Sameway answered only on this computer; using it from a phone meant
// setting up Tailscale, which a newcomer will not. Now the owner can let
// it answer on the home Wi-Fi too (internal/cli lan.go listens there), and
// a phone joins by scanning a code on Workspaces: a link with a pairing
// code that works once, for ten minutes, and leaves the phone a cookie
// that keeps it paired. On the Wi-Fi nothing but pairing answers anyone
// else, and a paired phone is the owner's own and is let in as them. The
// owner sees the phones paired and can take each away.

// lanCookie names the cookie a paired phone keeps.
const lanCookie = "sameway_device"

// lanPairFor is how long a pairing link works.
const lanPairFor = 10 * time.Minute

// lanDevice is a phone paired, by the hash of its cookie.
type lanDevice struct {
	ID    string    `json:"id"`
	Hash  string    `json:"hash"`
	Name  string    `json:"name"`
	Added time.Time `json:"added"`
	// Person is who it was invited for (lan_invite.go); none is the owner's.
	Person string `json:"person,omitempty"`
}

func token(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashOf(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (s *Server) lanDevices() []lanDevice {
	var out []lanDevice
	json.Unmarshal([]byte(s.app.Store.Meta("lan:devices")), &out)
	return out
}

func (s *Server) saveLanDevices(d []lanDevice) {
	b, _ := json.Marshal(d)
	s.app.Store.SetMeta("lan:devices", string(b))
}

// LAN is the handler the Wi-Fi listener serves: pairing, and the rest for
// a paired phone only, as the owner.
func (s *Server) LAN(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pair" && r.URL.Query().Has("invite") {
			s.lanJoin(w, r) // lan_invite.go
			return
		}
		if r.URL.Path == "/pair" {
			s.lanPair(w, r)
			return
		}
		if c, err := r.Cookie(lanCookie); err == nil {
			h := hashOf(c.Value)
			for _, d := range s.lanDevices() {
				if d.Hash != h {
					continue
				}
				v, ok := s.lanVisitor(d)
				if !ok {
					lanSay(w, "This device no longer opens this workspace. Ask its owner to invite you again.")
					return
				}
				next.ServeHTTP(w, r.WithContext(records.WithVisitor(r.Context(), v)))
				return
			}
		}
		lanPairPage(w, "This Sameway is private. To use it on this phone, open Workspaces in Sameway on the computer and choose Use Sameway on your phone. Then scan the code it shows, or type its letters here.") // phone_pair.go
	})
}

// lanPair is a phone opening a pairing link, or typing the letters: a
// code that works keeps the phone paired; one used, out of date or mistyped
// says to try the one shown now.
func (s *Server) lanPair(w http.ResponseWriter, r *http.Request) {
	if !s.pairing.use(r.URL.Query().Get("code"), time.Now()) {
		lanPairPage(w, "Those letters do not work: the code has been used, is more than ten minutes old, or was mistyped. Try the one Sameway shows now on the computer, under Workspaces.")
		return
	}
	secret := token(32)
	name := phoneName(r.UserAgent())
	devices := append(s.lanDevices(), lanDevice{ID: token(6), Hash: hashOf(secret), Name: name, Added: time.Now()})
	s.saveLanDevices(devices)
	records.Record(s.app.Store, "human", records.Change{Action: "paired", Component: "device", Detail: name})
	http.SetCookie(w, &http.Cookie{Name: lanCookie, Value: secret, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 400 * 24 * 3600})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// phoneName is what a device calls itself, in a word: an iPhone, an
// Android phone, a tablet.
func phoneName(ua string) string {
	switch l := strings.ToLower(ua); {
	case strings.Contains(l, "iphone"):
		return "iPhone"
	case strings.Contains(l, "ipad"):
		return "iPad"
	case strings.Contains(l, "android"):
		return "Android phone"
	case strings.Contains(l, "windows"), strings.Contains(l, "macintosh"), strings.Contains(l, "linux"), strings.Contains(l, "cros"):
		return "computer"
	}
	return "Phone"
}
