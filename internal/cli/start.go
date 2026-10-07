package cli

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Sameway with nothing after it is what a double-click runs, and a person
// who double-clicks a program expects it to open. It printed its help to a
// window that closed at once, and they saw a flash and nothing. Now it opens
// their workspace: the one they were standing in, else the one they used
// last, else a new one in their Documents folder, made the first time. One
// already running is opened in the browser, not started twice. sameway help
// still says the commands.

// startCmd opens the person's workspace, making it the first time.
func (c *ctx) startCmd() error {
	dir, made, err := c.yourWorkspace()
	if err != nil {
		return c.holdOpen(err)
	}
	if made {
		fmt.Fprintf(c.Stdout, "Made your workspace in %s.\n", dir)
	}
	// Already running: show it, do not start it again on a port it holds.
	for _, k := range workspace.KnownWorkspaces() {
		if k.Dir == dir && k.Addr != "" && answers("http://"+k.Addr) {
			fmt.Fprintf(c.Stdout, "Sameway is already open at http://%s/\n", k.Addr)
			if err := openInBrowser("http://" + k.Addr + "/"); err != nil {
				fmt.Fprintf(c.Stdout, "Visit http://%s/ in your browser.\n", k.Addr)
			}
			return nil
		}
	}
	keepProgram(c.Stdout) // keep.go
	fmt.Fprintln(c.Stdout, "Opening Sameway in your browser…")
	if apart, err := goApart(c, dir); apart || err != nil { // apart.go
		return c.holdOpen(err)
	}
	c.workspaceDir, c.args, c.plain = dir, nil, true
	return c.holdOpen(c.openCmd())
}

// yourWorkspace is the workspace a bare sameway opens: the one this folder
// is in, the one used last, or a new one in Documents, which made says.
func (c *ctx) yourWorkspace() (dir string, made bool, err error) {
	if c.workspaceDir != "" {
		return c.workspaceDir, false, nil
	}
	here := c.Dir
	if here == "" {
		here, _ = os.Getwd()
	}
	if found, err := workspace.Find(here); err == nil {
		return found, false, nil
	}
	if known := workspace.KnownWorkspaces(); len(known) > 0 {
		return known[0].Dir, false, nil
	}
	dir = defaultHome()
	// What init says next (sameway serve --workspace …) is for a terminal;
	// a double-click goes straight on to open it.
	out := c.Stdout
	c.args = []string{dir}
	c.Stdout = io.Discard
	err = c.initCmd()
	c.Stdout = out
	if err != nil {
		return "", false, err
	}
	return dir, true, nil
}

// defaultHome is where a first workspace goes: Sameway in the person's
// Documents folder, or in their home folder when they have none.
func defaultHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if docs := filepath.Join(home, "Documents"); isDir(docs) {
		return filepath.Join(docs, "Sameway")
	}
	return filepath.Join(home, "Sameway")
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// answers says whether a workspace server answers at base.
func answers(base string) bool {
	client := http.Client{Timeout: time.Second}
	res, err := client.Get(base + "/api/describe")
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode == http.StatusOK
}

// holdOpen keeps a double-clicked window open on an error until it is
// read: closed at once, it took the reason with it.
func (c *ctx) holdOpen(err error) error {
	if err == nil {
		return nil
	}
	fmt.Fprintf(c.Stderr, "\nSameway could not open: %v\n\nPress Enter to close this window.", err)
	if c.Stdin != nil {
		bufio.NewReader(c.Stdin).ReadString('\n')
	}
	return err
}

// nextFree listens on the first free port after addr's, of the next twenty.
func nextFree(addr string) (net.Listener, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	n, _ := strconv.Atoi(port)
	for p := n + 1; p <= n+20; p++ {
		if l, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(p))); err == nil {
			return l, nil
		}
	}
	return nil, fmt.Errorf("%s and the twenty ports after it are all in use", addr)
}
