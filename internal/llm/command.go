package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Command is a model reached through a program on this machine that the
// person is already signed in to: Claude Code, Codex, Gemini's CLI. No
// API key, because the program has the person's own session. Each turn
// runs the program once with the conversation as its prompt and this
// workspace's MCP server as its tools, and takes what it prints back.
// The tools run inside that program, so the receipt under the reply is
// read from the activity log rather than from tool calls seen here.
type Command struct {
	// Template is the command line, with {mcp} and {model} standing for
	// the path of an MCP configuration pointing at this workspace and the
	// model name, and {prompt} and {system} for the conversation and the
	// system prompt when a program wants them as arguments. Without those
	// two, both go to the program on stdin, which is what claude -p reads
	// and what a command line on Windows, capped at a few thousand
	// characters, cannot carry.
	Template string
	// Field is the JSON field holding the reply in what the program
	// prints, such as result; empty takes the whole output as the reply.
	Field      string
	Model      string
	Workspace  string
	Executable string
	Timeout    time.Duration
	Label      string
}

// Presets are the programs known well enough to fill the template in.
var Presets = map[string]Command{
	"claude-code": {Label: "Claude Code", Field: "result",
		Template: "claude -p --mcp-config {mcp} --allowedTools mcp__sameway__* --output-format json"},
}

func (c *Command) Name() string {
	if c.Label != "" {
		return c.Label
	}
	return "command"
}

// ToolsOutside says the program runs the tools itself, over MCP.
func (c *Command) ToolsOutside() bool { return true }

func (c *Command) Complete(ctx context.Context, req Request) (*Response, error) {
	if strings.TrimSpace(c.Template) == "" {
		return nil, errors.New("llm.command is empty: the command line to run, with {prompt} where the conversation goes")
	}
	mcpPath, err := c.mcpConfig()
	if err != nil {
		return nil, err
	}
	defer os.Remove(mcpPath)
	args := fill(tokens(c.Template), map[string]string{
		"{prompt}": transcript(req), "{system}": req.System, "{mcp}": mcpPath, "{model}": c.Model,
	})
	if len(args) == 0 {
		return nil, errors.New("llm.command names no program")
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if info, err := os.Stat(c.Workspace); err == nil && info.IsDir() {
		cmd.Dir = c.Workspace
	}
	cmd.Stdin = strings.NewReader(onStdin(c.Template, req))
	var out, errs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errs
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %v: %s", args[0], err, strings.TrimSpace(tail(errs.String(), 600)))
	}
	text := strings.TrimSpace(out.String())
	if c.Field != "" {
		var doc map[string]any
		if err := json.Unmarshal([]byte(text), &doc); err != nil {
			return nil, fmt.Errorf("%s printed something that is not JSON with a %s field: %s", args[0], c.Field, tail(text, 300))
		}
		v, _ := doc[c.Field].(string)
		text = strings.TrimSpace(v)
	}
	return &Response{Text: text}, nil
}

// mcpConfig writes the MCP configuration the program is handed: this
// binary, serving this workspace over stdio, under the name sameway.
func (c *Command) mcpConfig() (string, error) {
	exe := c.Executable
	if exe == "" {
		exe, _ = os.Executable()
	}
	f, err := os.CreateTemp("", "sameway-mcp-*.json")
	if err != nil {
		return "", err
	}
	cfg := map[string]any{"mcpServers": map[string]any{"sameway": map[string]any{"command": exe, "args": []string{"--workspace", c.Workspace, "mcp"}}}}
	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	return filepath.Clean(f.Name()), nil
}

// onStdin is what the program reads on its standard input: the system
// prompt and the conversation, each unless the template takes it as an
// argument, so the whole thing arrives however long it is.
func onStdin(template string, req Request) string {
	var b strings.Builder
	if !strings.Contains(template, "{system}") && req.System != "" {
		b.WriteString(req.System)
		b.WriteString("\n\n---\n\n")
	}
	if !strings.Contains(template, "{prompt}") {
		b.WriteString(transcript(req))
	}
	return b.String()
}

// transcript is the conversation as one prompt: what was said before,
// then what the person just said, which the program answers.
func transcript(req Request) string {
	var b strings.Builder
	for i, m := range req.Messages {
		if m.Content == "" && len(m.ToolCalls) == 0 {
			continue
		}
		last := i == len(req.Messages)-1
		switch m.Role {
		case RoleUser:
			if last {
				if b.Len() > 0 {
					b.WriteString("\n\nNow the person says:\n")
				}
				b.WriteString(m.Content)
			} else {
				b.WriteString("Person: " + m.Content + "\n")
			}
		case RoleAssistant:
			if m.Content != "" {
				b.WriteString("You: " + m.Content + "\n")
			}
			for _, tc := range m.ToolCalls {
				b.WriteString("You used " + tc.Name + "\n")
			}
		}
	}
	return b.String()
}

// tokens splits a command line the way a shell would for plain words and
// double-quoted phrases; a placeholder stays one token.
func tokens(line string) []string {
	var out []string
	var cur strings.Builder
	quoted, have := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted, have = !quoted, true
		case r == ' ' && !quoted:
			if have {
				out = append(out, cur.String())
				cur.Reset()
				have = false
			}
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if have {
		out = append(out, cur.String())
	}
	return out
}

func fill(args []string, values map[string]string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if v, ok := values[a]; ok {
			if a == "{model}" && v == "" {
				continue
			}
			out = append(out, v)
			continue
		}
		out = append(out, a)
	}
	return out
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
