# The command line

`sameway` in a terminal, for a person or an agent: the same records,
tools and words as the pages. The code is `internal/cli` (`root.go` has
the usage text, `help.go` the commands and what a typo meant); the tests
are `internal/cli/help_test.go`, `find_test.go` and `cli_test.go`.

## Why it works this way

- **Help is asked for, and answered on stdout.** `sameway help`, `-h`
  and `--help` say every command and exit 0. After a command,
  `sameway init --help` or `sameway help init` say only that command's
  lines, and the command does not run: `chat --help` would ask the model
  and `restore --help` would look for a copy named `--help`. The usage
  text is the one list; a test holds every command to a line of it
  ([clig.dev, Help](https://clig.dev/#help);
  [GNU Coding Standards, --help](https://www.gnu.org/prep/standards/html_node/_002d_002dhelp.html)).
- **A typo is answered with what was meant, not run.** `sameway descrbe`
  says `Did you mean sameway describe?`, as git and cargo do, for a
  command, a content type or a verb (two letters apart at most). It is
  never run for the person: they learn the right word
  ([clig.dev, Errors](https://clig.dev/#errors)).
- **Exit status says which kind of failure.** 0 done, 1 it went wrong,
  2 the command line was not understood (a usage line, an unknown
  command, verb or flag). 2 is what Go's flag package and the shell's
  own commands use for misuse, so a script or an agent can tell "fix the
  command" from "fix the data"
  ([pkg.go.dev/flag](https://pkg.go.dev/flag);
  [Bash, Exit Status](https://www.gnu.org/software/bash/manual/html_node/Exit-Status.html)).
  A flag alone (`sameway -x`) is misuse too: before, it was taken for no
  command and opened the workspace, as a double-click does.
- **Errors say what to do next, once, on stderr.** Outside a workspace a
  command names the ones this computer knows and how to pick one. A
  record not there says which, and `sameway note list`. A flag the
  command lacks is said once, by the flag package, with the command's
  flags. With `--json` an error is `{"error": "..."}` on stderr, the same
  words ([clig.dev, Output](https://clig.dev/#output);
  [Heroku CLI style guide](https://devcenter.heroku.com/articles/cli-style-guide)).
- **An agent reads the same thing as JSON.** Every command takes
  `--json`; an empty list is `[]`, never `null`, so a loop over it does
  not break.
- **Plain text a screen reader reads well.** No colour, no spinners, no
  box drawing: lines of words, a table only where it is one (describe,
  list). Nothing to turn off, so `NO_COLOR` holds already
  ([no-color.org](https://no-color.org/)).

## Not done, and why

- **No colour.** It would need `NO_COLOR`, TTY and `TERM=dumb` checks for
  little gain; the words carry the meaning.
- **No `--help --json`.** `sameway describe --json` is where an agent
  reads what a workspace offers; the commands do not change between
  workspaces.
- **A command's own flags are in its usage line, not their full
  descriptions.** Showing those would mean running the command's flag
  parsing, and for `chat`, `export` and `restore` that runs the command.
