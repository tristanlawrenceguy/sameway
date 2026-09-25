package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

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
	// Actions with a schedule run, and reminders ring, while the server does.
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.Chat.StartSchedule(ctx)
	h := server.New(a)
	h.StartRinging(ctx, notifier(a))
	keepSnapshots(ctx, c.Stdout, a)
	connectDevices(ctx, c.Stdout, a)
	watchUpdates(ctx, c.Stdout, a)
	token := os.Getenv(a.Workspace.Config.MCP.TokenEnv)
	if token != "" {
		fmt.Fprintf(c.Stdout, "  mcp     http://%s/mcp with Authorization: Bearer <%s>\n", *addr, a.Workspace.Config.MCP.TokenEnv)
	}
	all := HandlerFor(a, token, h)
	joinTailnet(ctx, c.Stdout, a, all, h)
	return http.ListenAndServe(*addr, all)
}

func (c *ctx) describeCmd() error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	d := a.Describe()
	if len(c.args) > 0 {
		// A part is read for its content, and that is JSON whether or not
		// --json was given: sameway describe types note.
		name := ""
		if len(c.args) > 1 {
			name = c.args[1]
		}
		v, err := d.Part(c.args[0], name)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
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
		fmt.Fprintln(c.Stdout, "\nAssistant tools:")
		for _, tool := range d.Tools {
			fmt.Fprintf(c.Stdout, "  %-16s %s\n", tool.Name, tool.Description)
		}
		fmt.Fprintln(c.Stdout, "\nRun with --json for schemas and manifests.")
	})
	return nil
}

// mcpCmd serves the workspace to one Model Context Protocol client on stdin
// and stdout, which is how MCP hosts start a server. Nothing else may be
// printed on stdout while it runs; anything for a person goes to stderr.
func (c *ctx) mcpCmd() error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	in := c.Stdin
	if in == nil {
		in = os.Stdin
	}
	fmt.Fprintf(c.Stderr, "sameway mcp: serving %q from %s\n", a.Workspace.Config.Name, a.Workspace.Dir)
	srv := &mcp.Server{App: a, Version: update.Version, In: in, Out: c.Stdout}
	return srv.Serve(context.Background())
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
