package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/peers"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/tailnet"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// joinTailnet serves h on the person's tailnet too, for as long as
// workspace.yaml names the machine there. The name is read as the server
// runs, so the assistant turning it on (or off) takes effect at once, with
// no restart. Every step goes to the terminal; while the person is setting
// it up, the steps also go to the chat, where they can follow the links.
//
// It is two background jobs: following the name every two seconds, and
// keeping the copies on the computers named in tailnet.peers in step every
// three.
func joinTailnet(out io.Writer, a *app.App, h http.Handler, srv *server.Server) []runner.Job {
	// Who gets in, and as whom, is the workspace's to say: its owner's own
	// devices, and the people it has let in.
	admit := func(ctx context.Context, p tailnet.Peer) (context.Context, string, bool) {
		return srv.Admit(ctx, p.Login, p.Name, p.Device, p.Owner)
	}
	// What the owner published is what the internet reaches, through
	// Funnel, with MCP for AI services when that is published too.
	pub := &tailnet.Funnel{
		Want: func() bool { return srv.Published().Any() },
		Handler: srv.Public(&mcp.Server{App: a, Version: update.Version,
			Published: func() map[string]bool { return srv.Published().Types }}),
	}
	n := &tailnetNode{out: out, a: a, h: h, admit: admit, pub: pub, published: publishing(a), news: make(chan struct{}), said: map[string]string{}}
	a.Chat.Tailnet = n.wait
	return []runner.Job{
		{Name: "Tailscale, as the workspace's settings say", Every: 2 * time.Second, Run: n.follow},
		{Name: "Copies on your other computers kept in step", Every: 3 * time.Second, Wait: true, Run: func(ctx context.Context, _ time.Time) error {
			return n.keepInStep(ctx, srv)
		}},
	}
}

type tailnetNode struct {
	out   io.Writer
	a     *app.App
	h     http.Handler
	admit tailnet.Admit
	pub   *tailnet.Funnel
	// published is what was published when the server started: being
	// published again at start is said on the terminal only.
	published string

	mu     sync.Mutex
	last   tailnet.Status
	client *http.Client  // reaches the other computers, once on the tailnet
	news   chan struct{} // closed when there is a new step, then replaced
	// setup is true from the name changing while the server runs, or a
	// step the person has to take, until the workspace is ready: the time
	// the chat hears each step. A workspace that simply starts up on its
	// tailnet says so on the terminal only.
	setup bool

	// What follow knows between its runs: the name it last joined as,
	// whether it has run, and how to leave.
	name    string
	started bool
	leave   context.CancelFunc
	// What keepInStep knows between its runs: whether the records from
	// before there were peers are stamped, and what each peer last said.
	seeded bool
	said   map[string]string
}

// follow joins, leaves or joins again when the name in the settings
// changed since it last looked. ctx is the runner's, so a node started
// here lasts until the program stops.
func (n *tailnetNode) follow(ctx context.Context, _ time.Time) error {
	want := strings.TrimSpace(n.a.Workspace.Config.Tailnet.Name)
	if want == n.name && n.started {
		return nil
	}
	first := !n.started
	if n.leave != nil {
		n.leave()
		n.leave = nil
	}
	if !first {
		n.mu.Lock()
		n.setup = true
		n.mu.Unlock()
	}
	if want == "" && !first {
		n.say(tailnet.Status{State: tailnet.Off})
	}
	n.name, n.started = want, true
	if want == "" {
		return nil
	}
	node, cancel := context.WithCancel(ctx)
	n.leave = cancel
	if err := tailnet.Start(node, n.a.Workspace.Config.Tailnet, n.h, n.admit, n.pub, n.say); err != nil {
		n.say(tailnet.Status{State: tailnet.Failed, Err: err})
		return err
	}
	return nil
}

func (n *tailnetNode) say(s tailnet.Status) {
	fmt.Fprintf(n.out, "  tailnet %s\n", s)
	n.mu.Lock()
	if s.Client != nil {
		n.client = s.Client
	}
	if s.OwnerLogin != "" {
		n.a.Records.Owner = records.Visitor{Access: records.Owner, Login: s.OwnerLogin, Name: s.OwnerName}
	}
	if s.State == tailnet.Off || s.State == tailnet.Failed {
		n.client = nil
	}
	if s.State == tailnet.SignIn || s.State == tailnet.NeedsHTTPS {
		n.setup = true
	}
	tell := n.setup
	// Publishing follows the owner's own ask, so what came of it is said
	// in the chat, unless it is the same as when the server started.
	switch s.State {
	case tailnet.Published, tailnet.Unpublished, tailnet.PublishFailed:
		tell = publishing(n.a) != n.published || s.State == tailnet.PublishFailed
		n.published = publishing(n.a)
	}
	if s.State == tailnet.Ready || s.State == tailnet.Off {
		n.setup = false
	}
	n.last = s
	close(n.news)
	n.news = make(chan struct{})
	n.mu.Unlock()
	if tell {
		n.a.Chat.Say(s.Words())
	}
}

// wait is the next step after it was asked, in the person's words, or
// the last one known if none comes in time: what set_setting answers when
// the assistant turns the tailnet on or off.
func (n *tailnetNode) wait(d time.Duration) string {
	n.mu.Lock()
	news := n.news
	n.mu.Unlock()
	select {
	case <-news:
	case <-time.After(d):
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.last.State == "" {
		return "Joining Tailscale; the next step will appear in the chat."
	}
	return n.last.Words()
}

// keepInStep keeps this copy of the workspace the same as the copies on the
// computers named in tailnet.peers, both ways, every few seconds, for as
// long as the server runs. Pages open here follow what arrives. Records
// from before there were peers are stamped once, so the first exchange
// carries them.
func (n *tailnetNode) keepInStep(ctx context.Context, srv *server.Server) error {
	list := peers.Peers(n.a.Workspace.Config.Tailnet.Peers)
	n.mu.Lock()
	client := n.client
	n.mu.Unlock()
	if client == nil || len(list) == 0 {
		return nil
	}
	if !n.seeded {
		if err := n.a.Store.Seed(); err != nil {
			fmt.Fprintf(n.out, "  sync    %v\n", err)
			return err
		}
		n.seeded = true
	}
	var failed []error
	for _, p := range list {
		step, cancel := context.WithTimeout(ctx, 20*time.Second)
		changed, there, err := peers.WithPresence(step, client, n.a.Store, p, srv.PresentHere())
		srv.HearPresence(there)
		cancel()
		if changed > 0 {
			srv.Changed()
		}
		// Each peer's state is said when it changes, not every round.
		now := "in step"
		if err != nil {
			now = err.Error()
			failed = append(failed, fmt.Errorf("%s: %w", p, err))
		}
		if n.said[p] != now {
			n.said[p] = now
			fmt.Fprintf(n.out, "  sync    %s: %s\n", p, now)
		}
	}
	return errors.Join(failed...)
}

// publishing is what the workspace publishes, as one line to compare.
func publishing(a *app.App) string {
	p := a.Workspace.Config.Publish
	return p.Tabs + "|" + p.Types
}
