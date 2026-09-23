// Package tailnet puts the workspace on the person's own Tailscale
// network, so a phone or another computer signed in to the same tailnet
// opens it from anywhere: no port forwarding, nothing public. The node
// runs inside this program (tsnet); nothing else needs installing on the
// machine that serves. Only the devices of the person who signed the node
// in may open it.
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
	// AuthKeyEnv names the environment variable holding a Tailscale auth
	// key, for a machine nobody signs in on. Without one, the first start
	// prints a link to sign in once; the node remembers it after that.
	AuthKeyEnv string `yaml:"auth_key_env"`
}

// Start joins the tailnet in the background and serves h there, on HTTPS
// when the tailnet has it turned on and on plain HTTP either way (the
// tailnet itself is encrypted, though browsers want https for ts.net). Only the devices of the person who signed
// it in get through, and mark tells each request which device it came
// from. say is told the sign-in link, the addresses, and anything that
// goes wrong; serving here never stops the workspace serving on this
// machine. It ends when ctx does.
func Start(ctx context.Context, cfg Config, h http.Handler, mark func(context.Context, string) context.Context, say func(string)) error {
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
				say("sign in to put this workspace on your tailnet: " + line[i+len("go to: "):])
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
	go serve(ctx, srv, h, mark, say)
	return nil
}

func serve(ctx context.Context, srv *tsnet.Server, h http.Handler, mark func(context.Context, string) context.Context, say func(string)) {
	st, err := srv.Up(ctx)
	if err != nil {
		if ctx.Err() == nil {
			say(fmt.Sprintf("not on the tailnet: %v", err))
		}
		return
	}
	host := strings.TrimSuffix(st.Self.DNSName, ".")
	lc, err := srv.LocalClient()
	if err != nil {
		say(fmt.Sprintf("not on the tailnet: %v", err))
		return
	}
	h = owner(lc, st.Self, mark, h)
	plain, err := srv.Listen("tcp", ":80")
	if err != nil {
		say(fmt.Sprintf("not on the tailnet: %v", err))
		return
	}
	go run(plain, h, say)
	// Browsers insist on https for ts.net addresses, so without it the
	// workspace cannot be opened. Turning it on is a switch in the
	// tailnet's admin console; keep trying so it starts once flipped.
	for said := false; ; said = true {
		secure, err := srv.ListenTLS("tcp", ":443")
		if err == nil {
			say(fmt.Sprintf("https://%s/", host))
			run(secure, h, say)
			return
		}
		if !said {
			say(fmt.Sprintf("https://%s/ needs HTTPS certificates turned on for your tailnet: https://login.tailscale.com/admin/dns (it starts here by itself once they are)", host))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func run(l net.Listener, h http.Handler, say func(string)) {
	if err := http.Serve(l, h); err != nil && !errors.Is(err, net.ErrClosed) {
		say(fmt.Sprintf("tailnet stopped: %v", err))
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
