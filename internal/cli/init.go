package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

func (c *ctx) initCmd() error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite an existing workspace's config and schema")
	noDetect := fs.Bool("no-detect", false, "do not probe for local model servers")
	positional, err := parseMixed(fs, c.args)
	if err != nil {
		return err
	}
	dir := ""
	if len(positional) > 0 {
		dir = positional[0]
	}
	// A folder given outright wins, then the global --workspace that every
	// other command takes, then where you are standing. Without the middle
	// one, `sameway --workspace elsewhere init` quietly built the workspace
	// in the current folder instead.
	if dir == "" {
		dir = c.workspaceDir
	}
	if dir == "" {
		dir = c.Dir
		if dir == "" {
			dir, _ = os.Getwd()
		}
	}
	abs, _ := filepath.Abs(dir)
	if err := workspace.Init(abs, examples.FS, examples.StarterRoot, *force); err != nil {
		return err
	}
	var found *llm.Detected
	signedIn := ""
	if !*noDetect {
		if hits := llm.Detect(context.Background(), llm.DefaultCandidates); len(hits) > 0 {
			found = &hits[0]
			if err := pointConfigAt(filepath.Join(abs, workspace.ConfigFile), found); err != nil {
				return err
			}
		} else if _, err := exec.LookPath("claude"); err == nil {
			// No model server, but Claude Code is here and signed in: the
			// chat can run through it, with no key to find.
			signedIn = "claude-code"
			if err := pointConfigAtCommand(filepath.Join(abs, workspace.ConfigFile), signedIn); err != nil {
				return err
			}
		}
	}
	c.print(map[string]any{"workspace": abs, "detected": found, "signed_in": signedIn}, func() {
		fmt.Fprintf(c.Stdout, "Created workspace in %s\n", abs)
		if signedIn != "" {
			fmt.Fprintf(c.Stdout, "No local model server answered, but Claude Code is installed: the chat will run through it, with your own sign-in and no API key.\n\nNext: sameway serve --workspace \"%s\"\n", abs)
			return
		}
		if found != nil {
			fmt.Fprintf(c.Stdout, "Found %s at %s and pointed the chat at model %q.\n", found.Server, found.BaseURL, found.Model)
			if len(found.Models) > 1 {
				fmt.Fprintf(c.Stdout, "Other models there: %s\n", strings.Join(found.Models[1:], ", "))
			}
			fmt.Fprintf(c.Stdout, "\nNext: sameway serve --workspace \"%s\"\n", abs)
			return
		}
		if !*noDetect {
			fmt.Fprintln(c.Stdout, "No local model server answered (tried Ollama, LM Studio, llama.cpp).")
		}
		fmt.Fprintf(c.Stdout, "\nNext:\n  1. Edit %s to point llm at your model.\n  2. Run: sameway serve --workspace \"%s\"\n", filepath.Join(abs, "workspace.yaml"), abs)
	})
	return nil
}

// pointConfigAtCommand points a fresh workspace.yaml at a program the
// person is signed in to, such as Claude Code, instead of a model server.
func pointConfigAtCommand(path, provider string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out := strings.Replace(string(src), "provider: openai\n", "provider: "+provider+"\n", 1)
	return os.WriteFile(path, []byte(out), 0o644)
}

// pointConfigAt rewrites the llm base_url and model lines of a freshly
// copied starter workspace.yaml so it talks to the detected server.
func pointConfigAt(path string, d *llm.Detected) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out := strings.Replace(string(src), "base_url: http://localhost:11434/v1", "base_url: "+d.BaseURL, 1)
	out = strings.Replace(out, "model: llama3.1", "model: "+d.Model, 1)
	return os.WriteFile(path, []byte(out), 0o644)
}
