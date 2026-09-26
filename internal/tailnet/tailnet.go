// Package tailnet puts the workspace on the person's own Tailscale
// network, so a phone or another computer signed in to the same tailnet
// opens it from anywhere: no port forwarding, nothing public. The node
// runs inside this program (tsnet); nothing else needs installing on the
// machine that serves. Who gets in is decided by the workspace; see Admit.
package tailnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tailscale.com/envknob"
	"tailscale.com/tsnet"
)

// Config is the tailnet section of workspace.yaml.
type Config struct {
	// Name is the machine name on the tailnet, which becomes the address:
	// name: home is https://home.<tailnet>.ts.net. Empty means off.
	Name string `yaml:"name"`
	// Peers are the other computers hosting this same workspace, by their
	// machine name on the tailnet, comma separated: this copy keeps in
	// step with each. See internal/sync.
	Peers string `yaml:"peers"`
	// AuthKeyEnv names the environment variable holding a Tailscale auth
	// key, for a machine nobody signs in on. Without one, the first start
	// prints a link to sign in once; the node remembers it after that.
	AuthKeyEnv string `yaml:"auth_key_env"`
}

// Start joins the tailnet in the background and serves h there, on HTTPS
// once the tailnet has certificates turned on and on plain HTTP for
// anything that is not a browser. admit decides who gets in (nil: only
// the person who signed it in) and marks each request with who they are. say is told each step (see Status); serving here never stops
// the workspace serving on this machine. It ends when ctx does.
func Start(ctx context.Context, cfg Config, h http.Handler, admit Admit, pub *Funnel, say func(Status)) error {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return nil
	}
	dir, err := stateDir(name)
	if err != nil {
		return err
	}
	// The node's logs stay on this machine rather than going to Tailscale.
	envknob.SetNoLogsNoSupport()
	// tsnet repeats the sign-in link every few seconds; say it once.
	var link string
	srv := &tsnet.Server{
		Hostname: name,
		Dir:      dir,
		UserLogf: func(format string, a ...any) {
			line := fmt.Sprintf(format, a...)
			if i := strings.Index(line, "go to: "); i >= 0 && line[i:] != link {
				link = line[i:]
				say(Status{State: SignIn, Link: line[i+len("go to: "):]})
			}
		},
	}
	if cfg.AuthKeyEnv != "" {
		srv.AuthKey = os.Getenv(cfg.AuthKeyEnv)
	}
	// A browser on the tailnet also visits other sites, and a page on one
	// of them must not be able to post forms here.
	h = http.NewCrossOriginProtection().Handler(h)
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	go serve(ctx, srv, h, admit, pub, say)
	return nil
}

func serve(ctx context.Context, srv *tsnet.Server, h http.Handler, admit Admit, pub *Funnel, say func(Status)) {
	st, err := srv.Up(ctx)
	if err != nil {
		if ctx.Err() == nil {
			say(Status{State: Failed, Err: err})
		}
		return
	}
	host := strings.TrimSuffix(st.Self.DNSName, ".")
	lc, err := srv.LocalClient()
	if err != nil {
		say(Status{State: Failed, Err: err})
		return
	}
	h = guard(lc, st.Self, admit, h)
	go publish(ctx, srv, host, pub, say)
	client := srv.HTTPClient()
	me := st.User[st.Self.UserID]
	plain, err := srv.Listen("tcp", ":80")
	if err != nil {
		say(Status{State: Failed, Err: err})
		return
	}
	go run(plain, h, say)
	// Browsers insist on https for ts.net addresses, so without it the
	// workspace cannot be opened. Turning it on is a switch in the
	// tailnet's admin console; keep trying so it starts once flipped.
	for said := false; ; said = true {
		secure, err := srv.ListenTLS("tcp", ":443")
		if err == nil {
			say(Status{State: Ready, Link: "https://" + host + "/", Client: client, OwnerLogin: me.LoginName, OwnerName: me.DisplayName})
			run(secure, h, say)
			return
		}
		if !said {
			say(Status{State: NeedsHTTPS, Link: HTTPSSettings, Host: host, Client: client, OwnerLogin: me.LoginName, OwnerName: me.DisplayName})
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func run(l net.Listener, h http.Handler, say func(Status)) {
	if err := http.Serve(l, h); err != nil && !errors.Is(err, net.ErrClosed) {
		say(Status{State: Failed, Err: err})
	}
}

// stateDir is where the node keeps its keys: under the user's config
// folder, never in the workspace, so they cannot end up in a git repo.
func stateDir(name string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "sameway", "tailnet", name)
	return dir, os.MkdirAll(dir, 0o700)
}
