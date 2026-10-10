# Working in this repository as an AI agent

This file is for you. Humans should read README.md and CONTRIBUTING.md.

## Run and verify

```bash
go run ./tools/check -base origin/main   # function and file size, debt list only goes down, component folders, tokens
go test ./...                 # includes golden output for every component
go vet ./... && gofmt -l .    # both must be clean
go build -o bin/sameway ./cmd/sameway
./bin/sameway --workspace examples/workspaces/starter describe --json
SAMEWAY_CLAUDE_CODE_MODEL=haiku go test ./internal/server -run ClaudeCode   # one real turn through Claude Code
SAMEWAY_MCP_LOG=calls.log ./bin/sameway serve   # each MCP tool a model calls, ok or refused and why, one line each
SAMEWAY_BENCH_MODEL=haiku SAMEWAY_BENCH_RUNS=3 go test ./internal/bench -timeout 140m -v   # the assistant measured on everyday requests
```

`make check` runs all of it. CI runs the same plus the Node accessibility
runner in tools/a11y-runner.

## Where things live

| Path | What | Read before editing |
|---|---|---|
| `design/components/<name>/` | one component: manifest, template, css, examples, README | `design/README.md` |
| `design/tokens/tokens.json` | design tokens; regenerate css with `go run ./tools/tokens` | |
| `design/base/*.css` | foundation, one concern per file, loaded in filename order | `design/foundations/` |
| `design/base/*.js` | the page scripts every page needs, joined in filename order before the components' enhance.js; `00-sw.js` is the core they share: `sw.arm(selector, fn)` instead of listening for the refresh, `sw.on`/`sw.emit`, `sw.status`, `sw.refresh`; `01-connect.js` is the one connection to the server, `sw.listen` for /events and `sw.stream` for a turn. What they do is tested in a browser by `tools/a11y-runner/behave.mjs` | `design/base/00-sw.js` |
| `internal/schema/` | content type files to Go types, validation, JSON Schema | `schema.go` |
| `internal/store/` | SQLite, one table per type | `store.go` |
| `internal/render/` | component registry, props validation, page layout | `registry.go` |
| `internal/ui/` | the components the pages use most as Go types (`ui.Button`, `ui.Link`, `ui.Alert`, `ui.TextField`, `ui.Empty`, `ui.Status`), rendered in the server with `s.part(...)`, and `ui.Form`, `s.form(...)`, the one way to write a form that does an action: hidden fields escaped, `from` and `back` under the names the server reads, its button a submit button. A test holds each type's fields to its manifest. Any other component stays a map with `s.component` | `ui.go`, `form.go` |
| `internal/blocks/` | the page model without HTTP: what a block shows, resolved from its props, the records and where it is shown; one Kind per component (resolve, what it shows, its noun, height, heading), used by the pages, the assistant and MCP alike | `kind.go`, `kinds.go` |
| `internal/relate/` | how one record connects to the others, worked out from the schema | `relate.go` |
| `internal/server/parts.go` | the parts of a page that are off until somebody asks: the keys, and who turned one on | `parts.go` |
| `internal/llm/` | provider-neutral chat + tools; openai.go and anthropic.go | `llm.go` |
| `internal/chat/` | the tool loop, and every operation the assistant, MCP and `POST /api/tools/{name}` offer, one `Op` each in one registry | `chat.go`, `op.go` |
| `internal/mcp/` | the Model Context Protocol server: the chat tools plus reading, over stdio | `server.go` |
| `internal/server/` | HTML pages and JSON API | `server.go`, `canvas.go` |
| `internal/web/` | what every handler shares: the route (who may use it, where what it changes is), the outcome a person is told and the way back to where they acted, a page's options, JSON answers, and `Deps`, the narrow face of the server a feature package's handlers are given | `route.go`, `deps.go` |
| `internal/server/media/` | files, recordings and meetings: a file kept and read, a picture and a recording on their pages, writing a recording down on this computer and telling its speakers apart, transcripts from Teams and Zoom. A feature package: its `Service` holds its state, its handlers get `media.Deps` (web.Deps and a few seams), its `Routes` go into the server's one route table (`server/features.go`) | `media.go`, `routes.go` |
| `internal/server/exchange/` | what comes in and goes out: records read from a file (CSV, contacts, a calendar, a mailbox), Bring your things, and a list, record, block or the whole workspace out as a spreadsheet, calendar, document or archive. A feature package like media, with `exchange.Deps` | `exchange.go`, `routes.go` |
| `internal/server/connect/` | connecting the assistant to an AI model: the card on the chat when there is none, each way of having one told as its own story, a pasted key checked and kept in the person's settings, a free model fetched through Ollama, the AI apps a person uses connected. A feature package that needs only `web.Deps` | `connect.go`, `routes.go` |
| `internal/server/servertest/` | what the server's tests and its feature packages' tests share: `New`/`NewWith` (an app on a fresh starter workspace and the server serving it), requests (`Get`, `PostForm`, `As`, `After`...), `Said`, and `Main` for a test package's TestMain | `servertest.go` |
| `internal/cli/` | the sameway command | `root.go` |
| `internal/runner/` | the work done in the background while a workspace is served: one `Runner` per app (`App.Jobs`), each `Job` a name, a schedule and a `Run(ctx, now)`; it waits on the injected clock, keeps each job's last run, last error and next run, recovers a panic, and waits for the jobs at shutdown. `cli/background.go` starts it for serve and open alike | `runner.go` |
| `internal/update/` | finding, checking and installing a release of sameway itself | `update.go` |
| `internal/bench/` | the assistant measured with a real model on everyday requests, each in a fresh workspace | `assistant_test.go` |
| `examples/workspaces/starter/` | what `sameway init` copies | |
| `design/brand/` | Sameway's icon in every form (svg, png, ico, icns, and the app's 192, 512 and square sizes); redraw with `go run ./tools/icons`; every page names /manifest.webmanifest (a standalone window, its icons) and a touch icon, so a browser installs Sameway as an app, which Help offers where it can (38-install.js) | `tools/icons/main.go`, `internal/server/app_install_test.go` |

## What the tests cover

The table of ways Sameway gets used, each with what is checked and where, is
split by area so parallel changes rarely touch the same file:

- [Components](docs/tests/components.md): each component on its own: rendered from props, read by its manifest, given hostile props, overridden, and seen, heard and used by keyboard in a real browser.
- [Canvas and tabs](docs/tests/canvas.md): the canvas the model builds: blocks, tabs, layout and its measuring, and blocks checked or set up wrong.
- [Pages people read](docs/tests/pages.md): the HTML pages a person reads and uses: what a page says and leaves out, lists narrowed and sorted, editing, and no machine words.
- [Outside agents](docs/tests/agents.md): agents that use Sameway from outside: the JSON API, MCP, the CLI, look, keys, and doing what a page does.
- [The assistant](docs/tests/assistant.md): the model inside Sameway: its tools, its prompt, how it is measured, and what it asks before doing.
- [People and computers together](docs/tests/together.md): who may open a workspace and how copies on several computers, a phone and the internet stay in step.
- [The program on a computer](docs/tests/program.md): starting, updating, first run, backups, undo, reminders and telling the makers.
- [Records in and out](docs/tests/data.md): records coming in from files and other apps, going out again, repeating, holding other records, writing, and actions that run on changes.
- [Recordings and meetings](docs/tests/recordings.md): recordings played, written down and said by who spoke, and meetings recorded and written up.

When you add a component, the contract and enum-coverage tests tell you what
is missing. When you add a way to use the system, add a row and a test to the
area it belongs to, or a new file in `docs/tests/` linked here.

## Rules the tooling enforces

- **Short functions, files split by topic.** `tools/check` fails a non-test
  Go function over 80 lines (count from `func` to its closing brace) and a
  source file over 300 lines. Both are hard limits. Long functions from
  before the rule are listed in `tools/check/debt.go`; that list only goes
  down: CI (`go run ./tools/check -base <base>`) fails a change that adds
  an entry or raises a length there, so shorten or split the function
  instead, and lower or remove its entry when you do. A new file named
  `*_more.go`, `*_extra.go` or `*_helpers.go` fails: when a file grows,
  split by topic, moving a group of related functions into a file named
  after what it holds. Read the whole file before editing.
- **Golden examples.** A component template must reproduce every example in
  its manifest byte for byte. After changing a template or manifest run
  `UPDATE_GOLDEN=1 go test ./internal/render/` and commit the example files.
- **Manifest completeness.** Every component needs props, a11y, machine, and
  examples sections plus README.md.
- **Tokens only.** Component CSS uses `var(--sw-...)`; no raw colours.

## Rules the tooling cannot enforce

- One concern per file. If you cannot say what a file does in one sentence, split it.
- Every CLI command supports `--json`. Every error says how to fix it.
- Accessibility is not optional: visible label, keyboard path, 44px target,
  7:1 contrast, no colour-only meaning. When unsure, use a native element.
  The browser suites in tools/a11y-runner enforce all five; a component
  that truly cannot meet one says why in `examples/a11y-waivers.json`.
- Do not add a frontend framework or client-side rendering. Pages are
  server-rendered HTML; progressive enhancement only, in a component's
  own `enhance.js`. Only what is not one component's (the core, the
  connection, following, the turn, editing) is in `design/base/*.js`,
  each file with a number of its own, in load order. Arm with `sw.arm`,
  never a listener for the refresh, and test what a script does in
  `tools/a11y-runner/behave-*.mjs`, not by reading its source.
- Do not add a dependency for something the standard library does.
- Tests run side by side: a new test starts with `t.Parallel()`. What an
  app takes from where it runs (the clock, this computer's folders, the
  webhook client, ntfy) comes in through `app.Options`
  (`newAppWith(t, app.Options{...})` in internal/server, `servertest.NewWith`), never a package
  variable or an environment variable. A test that must set one anyway
  leaves out `t.Parallel()` and puts it back before it ends.
- Motion explains a change and never moves focus. A person's own action
  moves in `motion-base` or less, transform and opacity only; under
  reduced motion and the still pace it may cross-fade but not travel
  (design/foundations/motion.md).
- New surfaces (MCP, export/import) are generated from the schema and
  manifests, never hand-written per type or component.

## What agents need: the checklist

The 22 checks from the agent accessibility audit (2026-09-28), each with where
it is checked, are in [docs/tests/agent-checklist.md](docs/tests/agent-checklist.md).

## Adding a component

```bash
./bin/sameway --workspace examples/workspaces/starter component new callout
```

Then fill in the manifest (props schema first), the template, the css, the
README, and add examples to the manifest. Run the golden update and
`go run ./tools/check`. Move the folder into `design/components/` when it is
meant to be built in.

## Adding a content type

Add `schema/<name>.yaml` to a workspace. A running server takes it within a
second (it looks at `schema/` on each request and every second, and open
pages follow); no restart and no code change needed.
If a field type is missing, add it in `internal/schema/validate.go`
(coerce + JSONSchema), `internal/store/crud.go` (encode/decode), and
`internal/server/views.go` (control), in that order, with a test.
