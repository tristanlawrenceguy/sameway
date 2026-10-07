package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/notify"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// openCmd is serve with the last step done for you: it starts the workspace
// and opens it in a browser. The difference matters to a person who has just
// cloned this and does not yet know there is a port to visit.
func (c *ctx) openCmd() error {
	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	addr := fs.String("addr", "", "listen address, overrides server.addr")
	page := fs.String("page", "/", "page to open, for example /design")
	stay := fs.Bool("no-browser", false, "start the workspace but do not open anything")
	if err := fs.Parse(c.args); err != nil {
		return err
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()

	asked := *addr != ""
	if *addr == "" {
		*addr = a.Workspace.Config.Server.Addr
	}
	// Bind before opening anything: a browser pointed at a port nobody is
	// listening on shows an error the person has to understand, and a port
	// already in use is worth saying plainly rather than racing on.
	listener, err := listenFor(*addr) // restart.go
	// The port in workspace.yaml is often another program's (8080 is many
	// a program's): the next free one will do, where none was asked for.
	if err != nil && !asked {
		listener, err = nextFree(*addr)
	}
	if err != nil {
		return fmt.Errorf("could not listen on %s: %w\nSomething else may already be using it; try --addr 127.0.0.1:8081", *addr, err)
	}
	where := "http://" + listener.Addr().String()

	// A double-click's window is read by a person who did not type a
	// command: where Sameway is and how to stop it, nothing for agents.
	if c.plain {
		fmt.Fprintf(c.Stdout, "Sameway is open at %s/\nYour workspace is in %s.\n\nKeep this window open while you use Sameway. Close it to stop Sameway.\n", where, a.Workspace.Dir)
	} else {
		fmt.Fprintf(c.Stdout, "sameway serving %q from %s\n  open    %s/\n  agents  %s/api/describe\n",
			a.Workspace.Config.Name, a.Workspace.Dir, where, where)
		if a.Chat.Provider == nil && a.Chat.ProviderErr != nil {
			fmt.Fprintf(c.Stdout, "  chat    disabled: %v\n", a.Chat.ProviderErr)
		}
	}
	if !*stay {
		target := where + "/" + strings.TrimPrefix(*page, "/")
		if err := openInBrowser(target); err != nil {
			fmt.Fprintf(c.Stdout, "  (could not open a browser: %v — visit %s yourself)\n", err, target)
		}
	}
	if !c.plain {
		fmt.Fprintln(c.Stdout, "\nPress Ctrl-C to stop.")
	}
	// This workspace is now one this machine knows, at this address, so
	// any other workspace can offer to open it. The page can start other
	// workspaces as servers of their own, and stop this one.
	workspace.Remember(a.Workspace.Dir, listener.Addr().String())
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	h := server.New(a)
	h.WriteDownInBackground()
	// MCP over HTTP too, as serve has it: at /mcp, for agents on this
	// computer with the workspace's token, and from the tailnet by who
	// Tailscale says they are, reading only.
	all := HandlerFor(a, os.Getenv(a.Workspace.Config.MCP.TokenEnv), h)
	var srv *http.Server
	srv = &http.Server{Handler: all}
	exit := func() {
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	lan := &lanServer{port: port, h: h.LAN(all)} // lan.go
	h.WithFleet(&server.Fleet{Launch: launchWorkspace, LAN: lan.set, LANBase: lan.Base, Exit: func() {
		go func() {
			time.Sleep(500 * time.Millisecond)
			srv.Shutdown(context.Background())
		}()
	}
	h.WithFleet(&server.Fleet{Launch: launchWorkspace, Exit: exit, Restart: func() error {
		if err := startAgain(a.Workspace.Dir, listener.Addr().String()); err != nil { // restart.go
			return err
		}
		exit()
		return nil
	}})
	// Reminders ring, scheduled actions run and the broker stays connected
	// for as long as the server does, with or without a page open.
	h.StartRinging(ctx, notifier(a))
	h.KeepCalendars(ctx) // calendars kept in step every hour
	a.Chat.StartSchedule(ctx)
	a.Chat.StartAutomating() // actions that run when something happens; chat/automate.go
	// Daily copies are kept either way; a double-click's window does not list them.
	notes := io.Writer(c.Stdout)
	if c.plain {
		notes = io.Discard
	}
	keepSnapshots(ctx, notes, a)
	// Kept current as serve is: a double-clicked Sameway, or one opened at
	// sign-in, never looked for a new version at all.
	watchUpdates(ctx, notes, a)
	connectDevices(ctx, c.Stdout, a)
	if a.Workspace.Config.Server.LAN == "on" {
		if err := lan.set(true); err != nil {
			fmt.Fprintf(c.Stdout, "  wi-fi   %v\n", err)
		}
	}
	joinTailnet(ctx, c.Stdout, a, all, h)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// notifier tells a ring beyond the page the way workspace.yaml says,
// read each time so a setting changed by asking applies at once.
func notifier(a *app.App) func(title, text, url string) {
	return func(title, text, url string) {
		cfg := a.Workspace.Config.Notify
		n := notify.Notifier{Desktop: cfg.Desktop != "off", Command: cfg.Command, Phone: cfg.Phone}
		if err := n.Send(title, text, url); err != nil {
			log.Printf("notify: %v", err)
		}
	}
}

// launchWorkspace starts this same program on another workspace, as a
// server of its own that outlives this one.
func launchWorkspace(dir, addr string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "open", "--workspace", dir, "--no-browser", "--addr", addr)
	return cmd.Start()
}

// openInBrowser hands a URL to whatever the operating system uses for one.
// Nothing is installed for this: every platform already has a way.
var openInBrowser = func(url string) error {
	switch runtime.GOOS {
	case "windows":
		// rundll32 takes the URL as a single argument, so a query string with
		// an ampersand in it does not have to survive a shell.
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// connectDevices reaches the MQTT broker workspace.yaml names, if any:
// subscribed topics become devices, and mqtt actions can publish.
func connectDevices(ctx context.Context, out io.Writer, a *app.App) {
	cfg := a.Workspace.Config.MQTT
	if strings.TrimSpace(cfg.Broker) == "" {
		return
	}
	bus, err := devices.Start(ctx, cfg, a.Store)
	if err != nil {
		fmt.Fprintf(out, "  mqtt    not connected: %v\n", err)
		log.Printf("devices: %v", err)
		return
	}
	a.Chat.Publish = bus.Publish
	fmt.Fprintf(out, "  mqtt    %s, %d topic(s)\n", cfg.Broker, len(cfg.Subscribe))
}
