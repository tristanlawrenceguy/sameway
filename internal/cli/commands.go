package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sameway-dev/sameway/examples"
	"github.com/sameway-dev/sameway/internal/server"
	"github.com/sameway-dev/sameway/internal/workspace"
)

func (c *ctx) initCmd() error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite an existing workspace's config and schema")
	if err := fs.Parse(c.args); err != nil {
		return err
	}
	dir := fs.Arg(0)
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
	c.print(map[string]any{"workspace": abs}, func() {
		fmt.Fprintf(c.Stdout, "Created workspace in %s\n\nNext:\n  1. Edit %s to point llm at your model.\n  2. Run: sameway serve --workspace \"%s\"\n", abs, filepath.Join(abs, "workspace.yaml"), abs)
	})
	return nil
}

func (c *ctx) serveCmd() error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "", "listen address, overrides server.addr")
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
	fmt.Fprintf(c.Stdout, "sameway serving %q from %s\n  open    http://%s/\n  agents  http://%s/api/describe\n", a.Workspace.Config.Name, a.Workspace.Dir, *addr, *addr)
	if a.Chat.Provider == nil && a.Chat.ProviderErr != nil {
		fmt.Fprintf(c.Stdout, "  chat    disabled: %v\n", a.Chat.ProviderErr)
	}
	return http.ListenAndServe(*addr, server.New(a))
}

func (c *ctx) describeCmd() error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	d := a.Describe()
	c.print(d, func() {
		fmt.Fprintf(c.Stdout, "Workspace: %s (%s)\n", d.Workspace, a.Workspace.Dir)
		fmt.Fprintf(c.Stdout, "Model:     %s %s ready=%v\n", d.LLM.Provider, d.LLM.Model, d.LLM.Ready)
		if d.LLM.Problem != "" {
			fmt.Fprintf(c.Stdout, "           %s\n", d.LLM.Problem)
		}
		fmt.Fprintln(c.Stdout, "\nContent types:")
		for _, t := range d.Types {
			names := make([]string, 0, len(t.Fields))
			for _, f := range t.Fields {
				names = append(names, f.Name+":"+f.Type)
			}
			fmt.Fprintf(c.Stdout, "  %-12s %3d records  %s\n", t.Name, t.Count, strings.Join(names, " "))
		}
		fmt.Fprintln(c.Stdout, "\nComponents:")
		for _, comp := range d.Components {
			fmt.Fprintf(c.Stdout, "  %-12s (%s) %s\n", comp.Name, comp.Source, comp.Description)
		}
		fmt.Fprintln(c.Stdout, "\nRun with --json for schemas and manifests.")
	})
	return nil
}

func (c *ctx) checkCmd() error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	problems := 0
	for _, comp := range a.Registry.Components() {
		for _, ex := range comp.Manifest.Examples {
			if _, err := comp.Render(ex.Props); err != nil {
				problems++
				fmt.Fprintf(c.Stderr, "component %s example %s: %v\n", comp.Manifest.Name, ex.Name, err)
			}
		}
	}
	if err := a.Chat.Available(); err != nil {
		fmt.Fprintf(c.Stderr, "chat: %v\n", err)
		problems++
	}
	if problems > 0 {
		return fmt.Errorf("%d problem(s) found", problems)
	}
	c.print(map[string]any{"ok": true, "types": len(a.Types.Types), "components": len(a.Registry.Components())}, func() {
		fmt.Fprintf(c.Stdout, "ok: %d content types, %d components\n", len(a.Types.Types), len(a.Registry.Components()))
	})
	return nil
}

func (c *ctx) chatCmd() error {
	text := strings.TrimSpace(strings.Join(c.args, " "))
	if text == "" {
		return errors.New("usage: sameway chat <message>")
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	rec, sendErr := a.Chat.Send(context.Background(), text)
	if rec == nil || (sendErr != nil && !c.JSON) {
		return sendErr
	}
	c.print(map[string]any{"reply": rec, "ok": sendErr == nil}, func() {
		fmt.Fprintln(c.Stdout, rec.Fields["content"])
	})
	return sendErr
}

func (c *ctx) componentCmd() error {
	if len(c.args) < 2 || c.args[0] != "new" {
		return errors.New("usage: sameway component new <name>")
	}
	name := c.args[1]
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	dir := filepath.Join(a.Workspace.ComponentsDir(), name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s already exists", dir)
	}
	if err := os.MkdirAll(filepath.Join(dir, "examples"), 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"manifest.json": strings.ReplaceAll(scaffoldManifest, "NAME", name),
		"template.html": strings.ReplaceAll(scaffoldTemplate, "NAME", name),
		"style.css":     ".sw-" + name + " { }\n",
		"README.md":     "# " + name + "\n\nWhen to use it, and when not to.\n",
	}
	for f, body := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(body), 0o644); err != nil {
			return err
		}
	}
	c.print(map[string]any{"component": name, "dir": dir}, func() {
		fmt.Fprintf(c.Stdout, "Scaffolded %s\nEdit manifest.json and template.html, then run `sameway check`.\n", dir)
	})
	return nil
}

const scaffoldManifest = `{
  "name": "NAME",
  "version": "0.1.0",
  "description": "What this component is for.",
  "props": {
    "type": "object",
    "additionalProperties": false,
    "required": ["text"],
    "properties": {
      "text": { "type": "string", "minLength": 1 },
      "id": { "type": "string" }
    }
  },
  "a11y": {
    "role": "",
    "keyboard": [],
    "states": [],
    "wcag": { "target": "AAA", "notes": "" }
  },
  "machine": {
    "selector": "[data-component=\"NAME\"]",
    "identify": "",
    "operate": ""
  },
  "examples": [
    { "name": "default", "props": { "text": "Hello" }, "file": "examples/default.html" }
  ]
}
`

const scaffoldTemplate = `<div class="sw-NAME" data-component="NAME"{{if .id}} id="{{.id}}"{{end}}>{{.text}}</div>
`
