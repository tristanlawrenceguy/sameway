package cli

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/server"
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

	if *addr == "" {
		*addr = a.Workspace.Config.Server.Addr
	}
	// Bind before opening anything: a browser pointed at a port nobody is
	// listening on shows an error the person has to understand, and a port
	// already in use is worth saying plainly rather than racing on.
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("could not listen on %s: %w\nSomething else may already be using it; try --addr 127.0.0.1:8081", *addr, err)
	}
	where := "http://" + listener.Addr().String()

	fmt.Fprintf(c.Stdout, "sameway serving %q from %s\n  open    %s/\n  agents  %s/api/describe\n",
		a.Workspace.Config.Name, a.Workspace.Dir, where, where)
	if a.Chat.Provider == nil && a.Chat.ProviderErr != nil {
		fmt.Fprintf(c.Stdout, "  chat    disabled: %v\n", a.Chat.ProviderErr)
	}
	if !*stay {
		target := where + "/" + strings.TrimPrefix(*page, "/")
		if err := openInBrowser(target); err != nil {
			fmt.Fprintf(c.Stdout, "  (could not open a browser: %v — visit %s yourself)\n", err, target)
		}
	}
	fmt.Fprintln(c.Stdout, "\nPress Ctrl-C to stop.")
	return http.Serve(listener, server.New(a))
}

// openInBrowser hands a URL to whatever the operating system uses for one.
// Nothing is installed for this: every platform already has a way.
func openInBrowser(url string) error {
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
