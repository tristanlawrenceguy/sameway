package tailnet

import (
	"context"
	"net/http"
	"strings"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// refused is what a device is told when nobody decides otherwise.
const refused = "This workspace opens only on devices signed in to Tailscale as the person who put it there.\n"

// Peer is who is at the other end of a request over the tailnet, as
// Tailscale knows them: their login (an email), the name they go by, the
// device, and whether they are the person who signed this node in.
type Peer struct {
	Login  string
	Name   string
	Device string
	Owner  bool
}

// Admit decides whether a peer gets in. It answers the request's context
// marked with who they are, or what to tell them when they do not. Nil
// lets in the owner alone.
type Admit func(ctx context.Context, p Peer) (context.Context, string, bool)

// guard asks Tailscale who each request comes from and lets Admit decide.
// A node signed in with a tagged auth key belongs to no person, so there
// everyone the tailnet's access rules let through counts as its owner.
func guard(lc *local.Client, self *ipnstate.PeerStatus, admit Admit, h http.Handler) http.Handler {
	tagged := self.Tags != nil && self.Tags.Len() > 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
		if err != nil || who.Node == nil {
			http.Error(w, refused, http.StatusForbidden)
			return
		}
		p := Peer{Device: deviceName(who.Node), Owner: ownersDevice(self.UserID, tagged, who.Node)}
		if who.UserProfile != nil && !who.Node.IsTagged() {
			p.Login, p.Name = who.UserProfile.LoginName, who.UserProfile.DisplayName
		}
		if admit == nil {
			if !p.Owner {
				http.Error(w, refused, http.StatusForbidden)
				return
			}
			h.ServeHTTP(w, r)
			return
		}
		ctx, say, ok := admit(r.Context(), p)
		if !ok {
			if say == "" {
				say = refused
			}
			http.Error(w, say, http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ownersDevice says whether a device is the owner's own: signed in as the same
// person as this node, or any the tailnet lets through when this node
// belongs to no person. A tagged device is never a person's.
func ownersDevice(me tailcfg.UserID, tagged bool, peer *tailcfg.Node) bool {
	if tagged {
		return true
	}
	return !peer.IsTagged() && peer.User == me
}

// deviceName is what the person calls the device: its name on the
// tailnet, such as pixel-7, without the tailnet's domain.
func deviceName(n *tailcfg.Node) string {
	if n.ComputedName != "" {
		return n.ComputedName
	}
	if name, _, _ := strings.Cut(n.Name, "."); name != "" {
		return name
	}
	return n.Hostinfo.Hostname()
}
