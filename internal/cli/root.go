// Package cli implements the sameway command. Every command supports --json
// so agents get structured output, and every error says how to fix it.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

const usage = `sameway - accessible content and components for people and agents

Usage:
  sameway                               open your workspace in the browser, making it the first time
  sameway init [dir] [--force] [--no-detect]  create a workspace from the starter preset
  sameway open [--addr host:port] [--page /path] [--no-browser]  run the workspace and open it in a browser
  sameway serve [--addr host:port]      run the web server for this workspace
  sameway describe [--json] [part [name]]  show content types, components, tools and routes, or one part
  sameway check                         validate the workspace schema and components
  sameway update [--check]              install a new version of sameway, or only say whether one is out
  sameway search <words>                find anything by the words in it, records and canvas blocks alike
  sameway add <file>... [--title words] add files on this computer, copied in as they are (up to 4 GB each)
  sameway agent add|list|remove <name>  let an agent in with a key of its own (--access view|edit|owner)
  sameway look <path> [--scripts] [--only controls] [--kind button] [--name words]  a page as a screen reader gets it, with its problems, as JSON
  sameway export [--zip file]           rewrite content/ from the database (it is kept current as things change)
  sameway import [--dry-run]            read content/ back into the database, after a git pull; one Undo takes it back
  sameway snapshots                     the daily copies of this workspace's data, newest first
  sameway restore <path>                put a copy back, keeping the data from before (workspace stopped)
  sameway chat <message>                talk to the assistant from the terminal
  sameway mcp [--assistant]             serve the workspace to an MCP client over stdio
  sameway connect <tool> [--write] [--config path]  the MCP configuration for claude-code, claude-desktop, cursor, windsurf, vscode, codex, or chatgpt for a client elsewhere
  sameway discover [type] [--wait 3s]   what the local network announces: brokers, hubs, speakers, printers
  sameway component new <name>          scaffold a component folder in the workspace
  sameway <type> list [--where field=value] [--order -field] [--limit n] [--format csv|xlsx|vcf|ics]  list records of a content type
  sameway <type> get <id> [--json]
  sameway <type> create --set k=v ... | --data '{json}'
  sameway <type> update <id> --set k=v ... | --data '{json}'
  sameway <type> delete <id>

Global flags:
  --workspace <dir>   workspace folder (default: nearest workspace.yaml, or $SAMEWAY_WORKSPACE)
  --json              machine-readable output
  --version
  -h, --help          these commands; after a command, its own lines

Exit status: 0 done, 1 it went wrong, 2 the command was not understood.
Errors go to stderr; with --json they are {"error": "..."}.
`

// Env carries the streams and working directory so tests can drive the CLI.
type Env struct {
	Stdout, Stderr io.Writer
	// Stdin is what `sameway mcp` reads its client from.
	Stdin io.Reader
	Dir   string
}

// Run executes args and returns the exit code: 0 done, 1 it went wrong,
// 2 the command line was not understood.
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
		fmt.Fprintln(env.Stdout, "sameway", update.Version)
		return 0
	}
	c := &ctx{Env: env, JSON: *jsonFlag, workspaceDir: *wsFlag, args: fs.Args()}
	var err error
	switch run, ok := commands[sub]; {
	case sub == "help":
		err = c.helpCmd()
	// A command's --help says its lines and does nothing: chat --help
	// would otherwise ask the model, and restore --help restore a copy.
	case sub != "" && wantsHelp(c.args):
		fmt.Fprint(c.Stdout, helpFor(sub))
	// Nothing after it is a double-click: open the person's workspace
	// (start.go). A flag alone is not nothing: sameway -x opened it.
	// (-psn_ is what an older macOS adds when an app is opened.)
	case sub == "" && len(c.args) > 0 && !strings.HasPrefix(c.args[0], "-psn_"):
		err = misused("sameway has no flag %s. The commands and flags: sameway help", c.args[0])
	case sub == "":
		if c.startCmd() != nil {
			return 1
		}
	case ok:
		err = run(c)
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
		// --help alone was no command, so it opened the workspace, as a
		// bare sameway does, instead of saying the commands.
		case sub == "" && (name == "help" || name == "h"):
			sub = "help"
		default:
			subArgs = append(subArgs, a)
		}
	}
	// "--" stops global parsing so subcommand flags like --set pass through.
	return sub, append(append(globals, "--"), subArgs...)
}

// parseMixed lets a subcommand take positional arguments before its flags,
// which is how people type commands (`init mydir --force`). It returns the
// leading positionals and parses the rest into fs.
func parseMixed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		positional = append(positional, arg)
	}
	return positional, fs.Parse(args[len(positional):])
}

type ctx struct {
	Env
	JSON         bool
	workspaceDir string
	args         []string
	app          *app.App
	// plain is a double-click's run: what open says is for a person, not a terminal.
	plain bool
}

// load opens the workspace for commands that need it.
func (c *ctx) load() (*app.App, error) {
	if c.app != nil {
		return c.app, nil
	}
	dir, err := c.workspaceHere() // find.go
	if err != nil {
		return nil, err
	}
	a, err := app.Load(dir, false)
	if err != nil {
		return nil, err
	}
	c.app = a
	return a, nil
}

// fail says what went wrong on stderr, as JSON with --json, and returns
// the exit code: 2 for a command line not understood, else 1.
func (c *ctx) fail(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0 // -help=true: the flag package has said the flags
	}
	code := 1
	misread, flagSaid := isUsage(err)
	if misread {
		code = 2
	}
	switch {
	case c.JSON:
		json.NewEncoder(c.Stderr).Encode(map[string]any{"error": err.Error()})
	case !flagSaid: // the flag package has said it, once is enough
		fmt.Fprintln(c.Stderr, "error:", err)
	}
	return code
}

func (c *ctx) print(v any, human func()) {
	if c.JSON {
		// An empty list is [] to an agent, not null.
		if r := reflect.ValueOf(v); r.Kind() == reflect.Slice && r.IsNil() {
			v = []any{}
		}
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(v)
		return
	}
	human()
}
