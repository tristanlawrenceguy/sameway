package cli

import (
	"flag"
	"fmt"
	"github.com/tristanlawrenceguy/sameway/internal/apps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// connect prints, or writes, the exact configuration that hooks a tool up
// to this workspace over MCP (internal/apps), for whoever uses a terminal;
// the page /apps writes the same for whoever does not.

func (c *ctx) connectCmd() error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	write := fs.Bool("write", false, "write the configuration into the tool's file (merging with what is there)")
	config := fs.String("config", "", "write to this file instead of the tool's usual one")
	positional, err := parseMixed(fs, c.args)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(apps.Apps))
	for k := range apps.Apps {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(positional) != 1 {
		return fmt.Errorf("usage: sameway connect <%s|chatgpt> [--write] [--config path]", strings.Join(names, "|"))
	}
	ws, err := c.resolveWorkspace()
	if err != nil {
		return err
	}
	if positional[0] == "chatgpt" || positional[0] == "remote" {
		fmt.Fprintf(c.Stdout, "A client elsewhere (ChatGPT's connectors, Claude's custom connectors, a hosted agent) reaches this workspace over HTTP:\n\n"+
			"  1. give it a key of its own: sameway agent add ChatGPT --access edit (or view, to read only); the key is shown once\n"+
			"  2. run: sameway serve, reachable to the client over HTTPS (a tunnel such as cloudflared or ngrok will do)\n"+
			"  3. give the client the URL https://<your host>/mcp with the header Authorization: Bearer <that key>\n\n"+
			"The log names it by its key, and sameway agent remove ChatGPT takes it away. The workspace's one\n"+
			"MCP token (SAMEWAY_MCP_TOKEN, or the variable named by mcp.token_env) still works, to read only.\n")
		return nil
	}
	cl, ok := apps.Apps[positional[0]]
	if !ok {
		return fmt.Errorf("no tool called %q; one of %s, or chatgpt for a client elsewhere", positional[0], strings.Join(names, ", "))
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "sameway"
	}
	server := apps.Server(cl, exe, ws)
	path := *config
	if path == "" {
		path = cl.Path(ws)
	}
	if !*write {
		fmt.Fprintf(c.Stdout, "Add this to %s (%s), or run again with --write:\n\n%s\n", path, cl.Note, apps.Snippet(cl, server))
		return nil
	}
	if err := apps.Write(path, cl, server); err != nil {
		return err
	}
	fmt.Fprintf(c.Stdout, "wrote %s (%s)\n", path, cl.Note)
	return nil
}

// resolveWorkspace is the workspace the configuration points at: the one
// given, or the one this directory is in, the way every command finds it.
func (c *ctx) resolveWorkspace() (string, error) {
	dir := c.workspaceDir
	if dir == "" {
		start := c.Dir
		if start == "" {
			start, _ = os.Getwd()
		}
		found, err := workspace.Find(start)
		if err != nil {
			return "", err
		}
		dir = found
	}
	return filepath.Abs(dir)
}
