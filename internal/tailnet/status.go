package tailnet

import (
	"fmt"
	"net/http"
)

// A State is where joining the tailnet has got to.
type State string

const (
	// SignIn: the node waits for the person to sign in to Tailscale once,
	// at Link.
	SignIn State = "sign-in"
	// NeedsHTTPS: the node is on the tailnet, but browsers insist on https
	// for ts.net addresses and the tailnet has no certificates yet; Link
	// is where they are turned on. It starts by itself once they are.
	NeedsHTTPS State = "needs-https"
	// Ready: the workspace opens at Link from the person's devices.
	Ready State = "ready"
	// Failed: the node could not start or stopped; Err says why.
	Failed State = "failed"
	// Off: no machine is named, so the workspace is on no tailnet.
	Off State = "off"
	// Published: what the owner published is readable by anyone on the
	// internet at Link, through Funnel.
	Published State = "published"
	// Unpublished: nothing is published any more.
	Unpublished State = "unpublished"
	// PublishFailed: Funnel would not open; Err says why, and it is tried
	// again by itself.
	PublishFailed State = "publish-failed"
)

// FunnelPolicy is where a tailnet's access policy is changed, to let a
// computer use Funnel.
const FunnelPolicy = "https://login.tailscale.com/admin/acls"

// HTTPSSettings is the page of the Tailscale admin console where a
// tailnet's HTTPS certificates are turned on.
const HTTPSSettings = "https://login.tailscale.com/admin/dns"

// Status is one step of joining, as it happens.
type Status struct {
	State State
	Link  string
	Host  string
	Err   error
	// Client reaches other machines on the tailnet, once the node is on
	// it (NeedsHTTPS and Ready): how this workspace talks to the other
	// computers that host it.
	Client *http.Client
	// OwnerLogin and OwnerName are who signed this node in, once it is on
	// the tailnet: who "You" is in what this computer writes down.
	OwnerLogin, OwnerName string
}

// String is the step as a line for the terminal the server runs in.
func (s Status) String() string {
	switch s.State {
	case SignIn:
		return "sign in to put this workspace on your tailnet: " + s.Link
	case NeedsHTTPS:
		return fmt.Sprintf("https://%s/ needs HTTPS certificates turned on for your tailnet: %s (it starts here by itself once they are)", s.Host, s.Link)
	case Ready:
		return s.Link
	case Off:
		return "off"
	case Published:
		return "published on the internet at " + s.Link
	case Unpublished:
		return "nothing published"
	case PublishFailed:
		return fmt.Sprintf("could not publish: %v", s.Err)
	}
	return fmt.Sprintf("not on the tailnet: %v", s.Err)
}

// Words is the step as the person is told it in the chat: what to do
// next, with the link to do it. Chat messages are plain text whose
// addresses become links, so the address stands on its own.
func (s Status) Words() string {
	switch s.State {
	case SignIn:
		return fmt.Sprintf("To open this workspace from your phone, sign in to Tailscale (a free account) here: %s\n\nThen install the Tailscale app on your phone and sign in there with the same account.", s.Link)
	case NeedsHTTPS:
		return fmt.Sprintf("One more step: turn on HTTPS certificates for your Tailscale network, with the Enable HTTPS button in Tailscale's DNS settings: %s\n\nBrowsers only open a Tailscale address over HTTPS. This workspace starts it by itself once you have.", s.Link)
	case Ready:
		return fmt.Sprintf("Your phone can open this workspace now. With the Tailscale app on, go to: %s\n\nOnly devices signed in to Tailscale as you get in.", s.Link)
	case Off:
		return "This workspace is no longer on your Tailscale network; your phone cannot open it until it is turned on again."
	case Published:
		return fmt.Sprintf("Published. Anyone can read what you published, with no login, at: %s\n\nEverything else stays private, and people on your tailnet still see the whole workspace at the same address.", s.Link)
	case Unpublished:
		return "Nothing is published any more; the public address shows nothing."
	case PublishFailed:
		return fmt.Sprintf("Publishing did not start: %v\n\nIf that is about Funnel, allow it for this computer in your Tailscale access policy, with the funnel attribute (Tailscale's default policy has it): %s\n\nIt tries again by itself.", s.Err, FunnelPolicy)
	}
	return fmt.Sprintf("This workspace could not join your Tailscale network: %v", s.Err)
}
