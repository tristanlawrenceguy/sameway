// Package apps hooks the AI apps a person already uses (Claude Desktop,
// Cursor, VS Code, Codex and others) up to their workspace over MCP: the
// exact configuration each wants, in the file it reads, merged with what
// is there. The command line prints or writes it (sameway connect); the
// page writes it for whoever never opens a terminal.
package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// App is one app that reads MCP servers from a file of its own.
type App struct {
	Name  string // as the person knows it
	Key   string // servers key in the file: mcpServers, or servers for VS Code
	Path  func(workspace string) string
	Shape string // json, vscode (adds type stdio) or toml
	Note  string // what to do after, in the app's own way
	// Here says whether it is on this computer, as far as can be told.
	Here func() bool
}

// Apps are the apps, by the name the command line takes.
var Apps = map[string]App{
	"claude-desktop": {"Claude Desktop", "mcpServers", func(string) string { return appData("Claude", "claude_desktop_config.json") }, "json",
		"Quit Claude Desktop fully (from its menu, not just the window) and open it again. Then ask it what is on your list today.",
		func() bool { return isDir(filepath.Dir(appData("Claude", "x"))) }},
	"claude-code": {"Claude Code", "mcpServers", func(ws string) string { return filepath.Join(ws, ".mcp.json") }, "json",
		"Open Claude Code in your workspace folder; it asks once whether to trust Sameway.",
		func() bool { _, err := exec.LookPath("claude"); return err == nil }},
	"cursor": {"Cursor", "mcpServers", func(string) string { return filepath.Join(home(), ".cursor", "mcp.json") }, "json",
		"Cursor finds it in Settings, MCP; switch Sameway on there if it is off.",
		func() bool { return isDir(filepath.Join(home(), ".cursor")) }},
	"windsurf": {"Windsurf", "mcpServers", func(string) string { return filepath.Join(home(), ".codeium", "windsurf", "mcp_config.json") }, "json",
		"Press refresh in Windsurf's MCP panel.",
		func() bool { return isDir(filepath.Join(home(), ".codeium", "windsurf")) }},
	"vscode": {"VS Code", "servers", func(ws string) string { return filepath.Join(ws, ".vscode", "mcp.json") }, "vscode",
		"Open your workspace folder in VS Code; it offers to start Sameway.",
		func() bool { _, err := exec.LookPath("code"); return err == nil }},
	"codex": {"Codex", "mcp_servers", func(string) string { return filepath.Join(home(), ".codex", "config.toml") }, "toml",
		"Start Codex again; it reads Sameway as it starts.",
		func() bool { return isDir(filepath.Join(home(), ".codex")) }},
}

// Server is how an app starts Sameway for a workspace: this program, with
// mcp.
func Server(app App, exe, workspace string) map[string]any {
	server := map[string]any{"command": exe, "args": []string{"--workspace", workspace, "mcp"}}
	if app.Shape == "vscode" {
		server["type"] = "stdio"
	}
	return server
}

// Snippet is the piece of the file the app needs, in the file's own shape.
func Snippet(app App, server map[string]any) string {
	if app.Shape == "toml" {
		args, _ := json.Marshal(server["args"])
		return fmt.Sprintf("[%s.sameway]\ncommand = %q\nargs = %s\n", app.Key, server["command"], args)
	}
	raw, _ := json.MarshalIndent(map[string]any{app.Key: map[string]any{"sameway": server}}, "", "  ")
	return string(raw)
}

// Write merges the sameway entry into the app's file, keeping every other
// server there, and makes the file when there is none.
func Write(path string, app App, server map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if app.Shape == "toml" {
		text := string(existing)
		head := "[" + app.Key + ".sameway]"
		if strings.Contains(text, head) {
			return fmt.Errorf("%s already has a %s section; edit it by hand or remove it and run again", path, head)
		}
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return os.WriteFile(path, []byte(text+"\n"+Snippet(app, server)), 0o644)
	}
	doc := map[string]any{}
	if strings.TrimSpace(string(existing)) != "" {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return fmt.Errorf("%s is not JSON I can add to: %w", path, err)
		}
	}
	servers, _ := doc[app.Key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["sameway"] = server
	doc[app.Key] = servers
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// Connected says whether an app's file already starts Sameway.
func Connected(app App, workspace string) bool {
	raw, err := os.ReadFile(app.Path(workspace))
	return err == nil && strings.Contains(string(raw), "sameway")
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
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
