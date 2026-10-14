package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
)

// commands are what sameway does by name; any other word is a content type
// (content.go). A test holds each to a line of the usage text.
var commands = map[string]func(*ctx) error{
	"init": (*ctx).initCmd, "open": (*ctx).openCmd, "serve": (*ctx).serveCmd,
	"at-login": (*ctx).atLoginCmd, // apart.go, internal/atlogin; started by the system, not typed
	"describe": (*ctx).describeCmd, "check": (*ctx).checkCmd, "update": (*ctx).updateCmd,
	"look": (*ctx).lookCmd, "add": (*ctx).addCmd, "search": (*ctx).searchCmd,
	"export": (*ctx).exportCmd, "import": (*ctx).importCmd, "snapshots": (*ctx).snapshotsCmd,
	"restore": (*ctx).restoreCmd, "chat": (*ctx).chatCmd, "connect": (*ctx).connectCmd,
	"agent": (*ctx).agentCmd, "discover": (*ctx).discoverCmd, "mcp": (*ctx).mcpCmd,
	"component": (*ctx).componentCmd,
}

// usageError is a command line sameway could not read: it exits 2, as Go's
// flag package and the shell's own commands do, so a script tells "you
// typed it wrong" from "it went wrong".
type usageError struct{ error }

func (e usageError) Unwrap() error { return e.error }

func misused(format string, a ...any) error { return usageError{fmt.Errorf(format, a...)} }

// isUsage is a usage error, said by sameway or by the flag package, which
// has already printed it with the command's flags.
func isUsage(err error) (usage, flagSaid bool) {
	if errors.As(err, new(usageError)) || strings.HasPrefix(err.Error(), "usage:") {
		return true, false
	}
	for _, p := range []string{"flag provided but not defined", "flag needs an argument", "bad flag syntax", "invalid value", "invalid boolean"} {
		if strings.HasPrefix(err.Error(), p) {
			return true, true
		}
	}
	return false, false
}

// wantsHelp is -h, -help or --help among a command's own arguments.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == "-h" || a == "-help" || a == "--help" {
			return true
		}
	}
	return false
}

// helpFor is the usage text's lines for one command, or for a content
// type's commands when cmd is not one of sameway's own.
func helpFor(cmd string) string {
	word := cmd
	if _, ok := commands[cmd]; !ok {
		word = "<type>"
	}
	var lines []string
	for _, l := range strings.Split(usage, "\n") {
		if f := strings.Fields(l); len(f) > 1 && f[0] == "sameway" && f[1] == word {
			lines = append(lines, l)
		}
	}
	if word == "<type>" {
		return fmt.Sprintf("%q is not a sameway command, so it is read as a content type (sameway describe types lists them):\n%s\n", cmd, strings.Join(lines, "\n"))
	}
	return "Usage:\n" + strings.Join(lines, "\n") + "\n\nGlobal flags: --workspace <dir>, --json. All the commands: sameway help\n"
}

// helpCmd says the commands, or one command: sameway help init.
func (c *ctx) helpCmd() error {
	if len(c.args) == 0 {
		fmt.Fprint(c.Stdout, usage)
		return nil
	}
	if _, ok := commands[c.args[0]]; !ok {
		if near := fold.Nearest(c.args[0], names()); near != "" {
			return misused("%q is not a command. Did you mean sameway help %s?", c.args[0], near)
		}
	}
	fmt.Fprint(c.Stdout, helpFor(c.args[0]))
	return nil
}

func names() []string {
	var out []string
	for n := range commands {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}
