package tailnet

import (
	"context"
	"net"
	"net/http"
	"time"

	"tailscale.com/tsnet"
)

// Funnel is what the workspace publishes to the internet, at its tailnet
// address, through Tailscale Funnel: Want says whether anything is
// published just now, and Handler answers the internet. The listener is
// Funnel's alone (FunnelOnly), so the internet only ever reaches Handler
// and the tailnet only ever reaches the workspace, at the same address.
type Funnel struct {
	Want    func() bool
	Handler http.Handler
}

// publish opens Funnel while something is published and closes it when
// nothing is, looking every few seconds, and says each step.
func publish(ctx context.Context, srv *tsnet.Server, host string, f *Funnel, say func(Status)) {
	if f == nil {
		return
	}
	var ln net.Listener
	failed := ""
	for {
		switch want := f.Want(); {
		case want && ln == nil:
			l, err := srv.ListenFunnel("tcp", ":443", tsnet.FunnelOnly())
			if err != nil {
				if err.Error() != failed {
					failed = err.Error()
					say(Status{State: PublishFailed, Err: err})
				}
				break
			}
			ln, failed = l, ""
			go http.Serve(l, f.Handler)
			say(Status{State: Published, Link: "https://" + host + "/"})
		case !want && ln != nil:
			ln.Close()
			ln = nil
			say(Status{State: Unpublished})
		}
		select {
		case <-ctx.Done():
			if ln != nil {
				ln.Close()
			}
			return
		case <-time.After(3 * time.Second):
		}
	}
}
