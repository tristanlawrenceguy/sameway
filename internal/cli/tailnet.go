package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/tailnet"
)

// joinTailnet serves h on the person's tailnet too, for as long as
// workspace.yaml names the machine there. The name is read as the server
// runs, so the assistant turning it on (or off) takes effect at once, with
// no restart. Every step goes to the terminal; while the person is setting
// it up, the steps also go to the chat, where they can follow the links.
func joinTailnet(ctx context.Context, out io.Writer, a *app.App, h http.Handler) {
	n := &tailnetNode{out: out, a: a, h: h, news: make(chan struct{})}
	a.Chat.Tailnet = n.wait
	go n.follow(ctx)
}

type tailnetNode struct {
	out io.Writer
	a   *app.App
	h   http.Handler

	mu   sync.Mutex
	last tailnet.Status
	news chan struct{} // closed when there is a new step, then replaced
	// setup is true from the name changing while the server runs, or a
	// step the person has to take, until the workspace is ready: the time
	// the chat hears each step. A workspace that simply starts up on its
	// tailnet says so on the terminal only.
	setup bool
}

func (n *tailnetNode) follow(ctx context.Context) {
	name, first := "", true
	var stop context.CancelFunc = func() {}
	defer func() { stop() }()
	for {
		if want := strings.TrimSpace(n.a.Workspace.Config.Tailnet.Name); want != name || first {
			stop()
			stop = func() {}
			if !first {
				n.mu.Lock()
				n.setup = true
				n.mu.Unlock()
			}
			if want == "" && !first {
				n.say(tailnet.Status{State: tailnet.Off})
			}
			if want != "" {
				node, cancel := context.WithCancel(ctx)
				stop = cancel
				if err := tailnet.Start(node, n.a.Workspace.Config.Tailnet, n.h, chat.WithVia, n.say); err != nil {
					n.say(tailnet.Status{State: tailnet.Failed, Err: err})
				}
			}
			name, first = want, false
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (n *tailnetNode) say(s tailnet.Status) {
	fmt.Fprintf(n.out, "  tailnet %s\n", s)
	n.mu.Lock()
	if s.State == tailnet.SignIn || s.State == tailnet.NeedsHTTPS {
		n.setup = true
	}
	tell := n.setup
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
