// Package cli implements the sameway command. Every command supports --json
// so agents get structured output, and every error says how to fix it.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Version is set by the build.
var Version = "dev"

const usage = `sameway - accessible content and components for people and agents

Usage:
  sameway init [dir] [--force]          create a workspace from the starter preset
  sameway serve [--addr host:port]      run the web server for this workspace
  sameway describe [--json]             show content types, components, and routes
  sameway check                         validate the workspace schema and components
  sameway chat <message>                talk to the assistant from the terminal
  sameway component new <name>          scaffold a component folder in the workspace
  sameway <type> list [--json]          list records of a content type
  sameway <type> get <id> [--json]
  sameway <type> create --set k=v ... | --data '{json}'
  sameway <type> update <id> --set k=v ... | --data '{json}'
  sameway <type> delete <id>

Global flags:
  --workspace <dir>   workspace folder (default: nearest workspace.yaml, or $SAMEWAY_WORKSPACE)
  --json              machine-readable output
  --version
`

// Env carries the streams and working directory so tests can drive the CLI.
type Env struct {
	Stdout, Stderr io.Writer
	Dir            string
}

// Run executes args and returns the exit code.
func Run(args []string, env Env) int {
	fs := flag.NewFlagSet("sameway", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	wsFlag := fs.String("workspace", os.Getenv("SAMEWAY_WORKSPACE"), "workspace folder")
	jsonFlag := fs.Bool("json", false, "machine-readable output")
	version := fs.Bool("version", false, "print version")
	fs.Usage = func() { fmt.Fprint(env.Stderr, usage) }
	// Allow global flags before or after the subcommand.
	sub, rest := splitGlobal(fs, args)
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(env.Stdout, "sameway", Version)
		return 0
	}
	if sub == "" || sub == "help" || sub == "-h" || sub == "--help" {
		fmt.Fprint(env.Stdout, usage)
		return 0
	}
	c := &ctx{Env: env, JSON: *jsonFlag, workspaceDir: *wsFlag, args: fs.Args()}
	var err error
	switch sub {
	case "init":
		err = c.initCmd()
	case "serve":
		err = c.serveCmd()
	case "describe":
		err = c.describeCmd()
	case "check":
		err = c.checkCmd()
	case "chat":
		err = c.chatCmd()
	case "component":
		err = c.componentCmd()
	default:
		err = c.contentCmd(sub)
	}
	if err != nil {
		return c.fail(err)
	}
	return 0
}

// splitGlobal separates global flags from the subcommand and its own args, so
// `--json` and `--workspace` may appear anywhere on the line. Globals come
// back first, followed by the subcommand's args, so one Parse call handles
// the globals and stops at the subcommand's first argument.
func splitGlobal(fs *flag.FlagSet, args []string) (string, []string) {
	var sub string
	var globals, subArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		name := strings.TrimLeft(a, "-")
		switch {
		case a == "--workspace" || a == "-workspace":
			globals = append(globals, a)
			if i+1 < len(args) {
				globals = append(globals, args[i+1])
				i++
			}
		case strings.HasPrefix(name, "workspace=") || name == "json" || name == "version":
			globals = append(globals, a)
		case sub == "" && !strings.HasPrefix(a, "-"):
			sub = a
		default:
			subArgs = append(subArgs, a)
		}
	}
	// "--" stops global parsing so subcommand flags like --set pass through.
	return sub, append(append(globals, "--"), subArgs...)
}

type ctx struct {
	Env
	JSON         bool
	workspaceDir string
	args         []string
	app          *app.App
}

// load opens the workspace for commands that need it.
func (c *ctx) load() (*app.App, error) {
	if c.app != nil {
		return c.app, nil
	}
	dir := c.workspaceDir
	if dir == "" {
		start := c.Dir
		if start == "" {
			start, _ = os.Getwd()
		}
		found, err := workspace.Find(start)
		if err != nil {
			return nil, err
		}
		dir = found
	}
	a, err := app.Load(dir, false)
	if err != nil {
		return nil, err
	}
	c.app = a
	return a, nil
}

func (c *ctx) fail(err error) int {
	if c.JSON {
		json.NewEncoder(c.Stderr).Encode(map[string]any{"error": err.Error()})
	} else {
		fmt.Fprintln(c.Stderr, "error:", err)
	}
	return 1
}

func (c *ctx) print(v any, human func()) {
	if c.JSON {
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(v)
		return
	}
	human()
}
