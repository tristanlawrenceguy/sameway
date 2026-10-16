package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// mcpConfig writes the MCP configuration the program is handed: this
// binary, serving this workspace over stdio, under the name sameway, as
// the assistant's, so what the program changes is logged as the
// assistant's and not as an outside agent's. It lists the tools the turn
// was given (chat/toolset.go), and more_tools for the rest, as the
// assistant in the app is given them.
func (c *Command) mcpConfig(req Request) (string, error) {
	exe := c.Executable
	if exe == "" {
		exe, _ = os.Executable()
	}
	f, err := os.CreateTemp("", "sameway-mcp-*.json")
	if err != nil {
		return "", err
	}
	args := []string{"--workspace", c.Workspace, "mcp", "--assistant"}
	var names []string
	for _, t := range req.Tools {
		if t.Name != "more_tools" { // the server makes its own, for what it holds back
			names = append(names, t.Name)
		}
	}
	if len(names) > 0 {
		args = append(args, "--tools", strings.Join(names, ","))
	}
	cfg := map[string]any{"mcpServers": map[string]any{"sameway": map[string]any{"command": exe, "args": args}}}
	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	return filepath.Clean(f.Name()), nil
}

// who names the program a person signed in to: Claude Code for claude.
func (c *Command) who(program string) string {
	base := strings.TrimSuffix(filepath.Base(program), filepath.Ext(program))
	if base == "claude" {
		return "Claude Code"
	}
	return base
}
