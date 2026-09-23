package tailnet

import (
	"context"
	"net/http"
	"strings"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// refused is what a device that is not the owner's is told.
const refused = "This workspace opens only on devices signed in to Tailscale as the person who put it there.\n"

// owner lets in the devices of the person who signed this node in, and
// marks each request with the device's name so what it changes says where
// it came from. A node signed in with a tagged auth key belongs to no
// person, so who gets in is left to the tailnet's access rules.
func owner(lc *local.Client, self *ipnstate.PeerStatus, mark func(context.Context, string) context.Context, h http.Handler) http.Handler {
	tagged := self.Tags != nil && self.Tags.Len() > 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
		if err != nil || who.Node == nil || !admit(self.UserID, tagged, who.Node) {
			http.Error(w, refused, http.StatusForbidden)
			return
		}
		if mark != nil {
			r = r.WithContext(mark(r.Context(), deviceName(who.Node)))
		}
		h.ServeHTTP(w, r)
	})
}

// admit says whether a device on the tailnet may open the workspace: one
// signed in as the same person as this node, or any the tailnet lets
// through when this node belongs to no person. A tagged device is never
// the person's own.
func admit(me tailcfg.UserID, tagged bool, peer *tailcfg.Node) bool {
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
