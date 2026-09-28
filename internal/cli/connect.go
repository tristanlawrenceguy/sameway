package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// connect prints, or writes, the exact configuration that hooks a tool up
// to this workspace over MCP. Every tool wants the same three facts, the
// program, its arguments and a name, in a file of its own, and nobody
// should have to know where that file is or what shape it takes.
type client struct {
	Key   string // servers key in the file: mcpServers, or servers for VS Code
	Path  func(workspace string) string
	Shape string // json, vscode (adds type stdio) or toml
	Note  string
}

var clients = map[string]client{
	"claude-code":    {"mcpServers", func(ws string) string { return filepath.Join(ws, ".mcp.json") }, "json", "project scope: the file sits in the workspace and travels with it in git; Claude Code asks once whether to trust it"},
	"claude-desktop": {"mcpServers", func(string) string { return appData("Claude", "claude_desktop_config.json") }, "json", "restart Claude Desktop afterwards"},
	"cursor":         {"mcpServers", func(string) string { return filepath.Join(home(), ".cursor", "mcp.json") }, "json", "or put the same in <project>/.cursor/mcp.json for one project"},
	"windsurf":       {"mcpServers", func(string) string { return filepath.Join(home(), ".codeium", "windsurf", "mcp_config.json") }, "json", "Windsurf reads it on the next refresh of its MCP panel"},
	"vscode":         {"servers", func(ws string) string { return filepath.Join(ws, ".vscode", "mcp.json") }, "vscode", "workspace scope, next to the workspace; VS Code offers to start the server"},
	"codex":          {"mcp_servers", func(string) string { return filepath.Join(home(), ".codex", "config.toml") }, "toml", "Codex CLI reads it at start"},
}

func (c *ctx) connectCmd() error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	write := fs.Bool("write", false, "write the configuration into the tool's file (merging with what is there)")
	config := fs.String("config", "", "write to this file instead of the tool's usual one")
	positional, err := parseMixed(fs, c.args)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(clients))
	for k := range clients {
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
			"  1. set SAMEWAY_MCP_TOKEN to a long secret (or the variable named by mcp.token_env in workspace.yaml)\n"+
			"  2. run: sameway serve, reachable to the client over HTTPS (a tunnel such as cloudflared or ngrok will do)\n"+
			"  3. give the client the URL https://<your host>/mcp with the header Authorization: Bearer <that secret>\n\n"+
			"Without the token the /mcp route answers 403 and says so.\n")
		return nil
	}
	cl, ok := clients[positional[0]]
	if !ok {
		return fmt.Errorf("no tool called %q; one of %s, or chatgpt for a client elsewhere", positional[0], strings.Join(names, ", "))
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "sameway"
	}
	server := map[string]any{"command": exe, "args": []string{"--workspace", ws, "mcp"}}
	if cl.Shape == "vscode" {
		server["type"] = "stdio"
	}
	path := *config
	if path == "" {
		path = cl.Path(ws)
	}
	if !*write {
		fmt.Fprintf(c.Stdout, "Add this to %s (%s), or run again with --write:\n\n%s\n", path, cl.Note, snippet(cl, server))
		return nil
	}
	if err := writeConfig(path, cl, server); err != nil {
		return err
	}
	fmt.Fprintf(c.Stdout, "wrote %s (%s)\n", path, cl.Note)
	return nil
}

// snippet is the piece of the file the tool needs, in the file's own shape.
func snippet(cl client, server map[string]any) string {
	if cl.Shape == "toml" {
		args, _ := json.Marshal(server["args"])
		return fmt.Sprintf("[%s.sameway]\ncommand = %q\nargs = %s\n", cl.Key, server["command"], args)
	}
	raw, _ := json.MarshalIndent(map[string]any{cl.Key: map[string]any{"sameway": server}}, "", "  ")
	return string(raw)
}

// writeConfig merges the sameway entry into the tool's file, keeping every
// other server there, and makes the file when there is none.
func writeConfig(path string, cl client, server map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if cl.Shape == "toml" {
		text := string(existing)
		head := "[" + cl.Key + ".sameway]"
		if strings.Contains(text, head) {
			return fmt.Errorf("%s already has a %s section; edit it by hand or remove it and run again", path, head)
		}
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return os.WriteFile(path, []byte(text+"\n"+snippet(cl, server)), 0o644)
	}
	doc := map[string]any{}
	if strings.TrimSpace(string(existing)) != "" {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return fmt.Errorf("%s is not JSON I can add to: %w", path, err)
		}
	}
	servers, _ := doc[cl.Key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["sameway"] = server
	doc[cl.Key] = servers
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
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

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

// appData is where an app keeps its settings on this platform.
func appData(app, file string) string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), app, file)
	case "darwin":
		return filepath.Join(home(), "Library", "Application Support", app, file)
	}
	return filepath.Join(home(), ".config", app, file)
}
