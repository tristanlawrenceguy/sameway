# Architecture

Name: **Sameway**. License: MIT. Module path: `github.com/tristanlawrenceguy/sameway`
(the account is tristanlawrenceguy; change the
module path in `go.mod` and the imports if the repo lands elsewhere).

Status: in daily use (section 8 says what is built). The home page is a
canvas the assistant builds through tools, beside the conversation; see
section 5.

A single Go binary that serves a structured-content system built on an accessible
design system. One contract per component and per content type generates every
surface: rendered HTML for humans, MCP tools and a JSON API for agents, and CLI
commands for both. Runs locally with zero setup. A workspace is a plain folder that
travels through git.

## 1. Decisions and why

| Decision | Choice | Reason |
|---|---|---|
| Backend language | Go | 1 to 3 second compile loop, one idiomatic style, gofmt/vet keep AI-written code uniform, pure-Go SQLite means no C toolchain, single binary. |
| Rendering | Server-rendered HTML, progressive enhancement, no frontend framework | Best accessibility by default. Output is plain HTML an agent can read as easily as a screen reader. Fits "documents, forms, lists". |
| Templates | Go `html/template`; props are a JSON object (`map[string]any`) | Stdlib, auto-escaping, no code generator. Every render validates the props against the manifest's JSON Schema and applies its defaults before the template runs (`internal/render`), so the manifest is the only definition of a component's props. |
| Storage | SQLite (modernc.org/sqlite) as live store, Markdown/JSON files as portable form | Zero setup. Files make the workspace git-friendly and AI-readable. `sameway export` and `sameway import` move between them. |
| CMS scope | Structured content types only, no page builder | Keep the core small. Rendering is done by components; if a view does not exist, create a component. |
| Connections | Worked out from the schema in `internal/relate`; a page shows none of them, an agent is given all of them | A connection is real in the data whether or not it is drawn. A page that opens every one is a page of other records with the one you came for at the top; a row of links to them is the same page in miniature, there every time for the once it is wanted. So the page shows what the record is, the whole graph goes to the API and the assistant, and `?show=<key>` opens the one there is a reason to open. |
| Design system | Standalone package (tokens, CSS, HTML patterns, manifests) | Usable in any stack. The Go binary is one consumer. |
| Accessibility bar | WCAG 2.2 AAA where feasible, AA as hard gate in CI | Every component ships with automated and keyboard tests. |
| Users | One owner, local first; other people by who Tailscale says they are | No accounts or passwords: the owner is whoever is at the computer or signed the node in, and other people get the access their `person` record gives (section 4). |
| Sharing | Workspace = git-friendly folder | Push a repo or sync a folder. Presets are just repos. |

## 2. The one contract

Everything an agent or a human needs is derived from two kinds of declarative files.

### 2.1 Component manifest

Each component is one folder. The manifest is the source of truth; the template
must render exactly what the examples show (enforced by golden tests).

```
design/components/button/
  manifest.json      # name, description, props schema, a11y contract, machine notes
  template.html      # html/template, receives props validated against the manifest
  style.css          # scoped by class prefix, uses tokens only
  enhance.js         # optional, progressive enhancement, no framework
  examples/          # plain HTML files, the standalone spec and golden output
  README.md          # when to use it, why it works this way, what is not done
```

`manifest.json` shape:

```json
{
  "name": "button",
  "version": "1.0.0",
  "description": "Triggers an action. Use link for navigation.",
  "props": { "$schema": "...", "type": "object", "properties": { "label": {...}, "variant": {...} }, "required": ["label"] },
  "a11y": {
    "role": "button",
    "keyboard": [ { "key": "Enter", "does": "activates" }, { "key": "Space", "does": "activates" } ],
    "states": [ "aria-pressed", "aria-disabled" ],
    "wcag": { "target": "AAA", "notes": "44x44 min target, 7:1 contrast on default variant" }
  },
  "machine": {
    "selector": "[data-component=button]",
    "identify": "data-label attribute or visible text",
    "operate": "click, or press Enter when focused"
  },
  "examples": [ { "name": "default", "props": { "label": "Save" }, "file": "examples/default.html" } ]
}
```

The `a11y` and `machine` blocks are the point of the project: the same landmarks,
roles, names, and keyboard map serve screen readers and browser-driving agents.

### 2.2 Content type schema

Content types live in the workspace, one file each.

```yaml
# workspace/schema/note.yaml
name: note
description: A short piece of writing
fields:
  title:   { type: string, required: true, maxLength: 200 }
  body:    { type: markdown }
  tags:    { type: list, of: string }
  status:  { type: enum, values: [draft, published], default: draft }
views:
  list:   { component: table, columns: [title, status, updatedAt] }
  detail: { component: article, title: title, body: body }
  form:   { component: form }
```

From one schema file the system generates:

- SQLite table and migration
- validation
- CLI: `sameway note create|get|list|update|delete`, all with `--json`
- Assistant and MCP tools: `create_record`, `update_record`, `find_records`,
  `get_record`, which take the type and check its fields
- JSON API: `/api/note`, `/api/note/{id}`
- HTML views: `/t/note`, `/t/note/{id}`, built from the design system's components
- `describe` output so an agent can learn the type without reading rows

Nothing is hand-written per surface. Adding a surface means adding a generator.

A few facts about a record are asked everywhere, so the schema answers
each once (`internal/schema/day.go`, `labels.go`, `called.go`) and no
surface decides for itself:

- **Its day** (`DayField`): `starts` when it is a shown datetime field,
  else the first shown datetime field. A list's groups, the calendar, a
  record's related things on its day, the calendar export, an import's
  date column and what a repeat moves all go by it. A hidden datetime is
  bookkeeping, not a day.
- **Whether it is done** (`DoneField`, `Done`): a yes-or-no named done,
  completed, complete or finished, or a pick-list at done (a reminder's
  state).
- **A field's name** (`Display`, and `Words` inside a sentence): its
  label, else its name with spaces.
- **A record's name** (`Called`, and `chat.Name` above it): its title,
  else the first thing it says, else its kind and id.
- **What is in it** (`listed: true` on a ref field): the records whose
  ref points at it, counted at a glance ("3 tasks, 1 done") and listed
  on its page under their own heading: a project's tasks, a meeting's
  tasks, a person's interactions in the starter schema. Any other ref is
  a connection a page opens when asked (`?show=`), never both.

## 3. Repository layout

```
/
  README.md  LICENSE  ARCHITECTURE.md  CONTRIBUTING.md  AGENTS.md
  cmd/sameway/main.go         # entry point
  internal/
    app/                      # wires a workspace, schema, store, components and chat
    workspace/                # the workspace folder: find, load, init, snapshots, trash
    schema/                   # content type files: load, validate, JSON Schema
    store/                    # SQLite, one table per type, field stamps for sync
    records/                  # writes, the activity log, undo, names, who asks and who wrote what
    render/                   # component registry, props validation, page layout
    relate/  query/  search/  # connections, the "which records" grammar, search
    when/  track/  trim/      # days and repeats, habit arithmetic, short titles
    prose/                    # Markdown rendered for the design system
    chat/                     # the conversation with a model: turns, prompts, tools
    llm/                      # provider-neutral chat: OpenAI-compatible and Anthropic
    server/                   # HTTP: HTML pages, JSON API, describe, look
    mcp/                      # MCP over stdio and HTTP, from the chat tools
    look/                     # reads a page as a screen reader or an agent does
    cli/                      # the sameway command, every command with --json
    content/  export/         # Markdown with front matter; CSV, Excel, vCard, iCal out
    ingest/  convert/         # files and other apps' exports in; files as text
    speech/  soundtrack/      # speech-to-text on this computer; sound out of video
    meetfetch/                # meeting transcripts from Teams and Zoom
    notify/  atlogin/         # notifications; opening at sign-in
    tailnet/  peers/          # Tailscale; keeping several hosts' copies the same
    devices/  discover/       # MQTT devices; what the local network announces
    update/                   # find, verify and install a release of sameway itself
    tokens/                   # tokens.json to CSS
    bench/                    # the assistant measured with a real model (tests only)
  design/                     # tokens, base, components, foundations, arrangements,
                              # brand; see design/README.md
  tools/
    check/                    # function and file size, component folders, tokens
    tokens/  icons/           # regenerate tokens.css; redraw the icon
    a11y-runner/              # Node dev-only: playwright + axe-core, pages and site
  docs/tests/                 # what each test covers, by area
  examples/workspaces/starter # what `sameway init` copies
```

`internal/records` is the workspace's records as Sameway keeps them,
whoever asks: who is asking and what they may do (`Visitor`), how a
record is written and logged (`WriteAs`, `Change`, `Record`), a change
as the ops it wrote, each thing as it was and became, written all or
none (`Op`, `Book.Apply`), undoing a change (`Book.Undo`: its ops the
other way; entries from before ops are read in undo_legacy.go), who
hears of a change (`records.Listen`, once per change logged;
`store.Listen`, once per record written, which automations use), what a
record and a change are called (`Name`,
`Sentence`), who wrote a record's words (`Writers`, `RecordView`) and an
agent key's pace. It sits on `store` and `schema`; the server, MCP, the
command line and `internal/chat` all go through it, so every way in
writes, logs, names and takes back the same way. `internal/chat` is the
conversation with a model (turns, prompts, tools), built on a
`records.Book`.

## 4. Workspace folder

```
my-workspace/
  workspace.yaml     # name, llm, server, ui, tailnet, publish and other settings
  schema/            # content types
  components/        # local components or overrides, same folder contract as design/
  content/           # exported records: content/note/<id>.md with frontmatter
  data.db            # live SQLite store, gitignored, refilled by `import`
```

`sameway init` creates one from the starter.
Multi-device is git or any folder sync. A phone or another computer can also
open a running workspace from anywhere over the person's Tailscale network
(`tailnet:` in workspace.yaml; the node is embedded with tsnet, its keys kept
under the user's config folder, never in the workspace). Only devices signed
in as the node's owner get in, and changes made from one carry its name in
the activity log (`via`).

### People and access

Other people reach a workspace the same way: over Tailscale, either on the
same tailnet (a team's) or with the machine shared to them from the owner's.
Tailscale says who each visitor is, by login, which is an email; sameway
says what they may do, by matching that email to a `person` record.

- **Owner**: whoever signed the node in, and anyone on the machine itself.
  Everything, including settings, the Workspaces page, and answering the
  assistant's questions.
- **Edit** (`person.access: edit`): content, the canvas, actions.
- **View** (`person.access: view`): reading only.
- Anyone else is refused, and the owner is asked in the chat whether to
  let them look. No forms: the owner can also tell the assistant ("let Bob
  edit"), which asks first; taking access away is immediate and not asked.

`access` is a field Sameway keeps: no page, API call or assistant tool
writes it except through that question, so nobody raises their own level.
What someone changes is logged under their name and device ("Bob removed
card Shopping, on pixel-7"). Everyone who may edit has their own chats with
the assistant (a conversation carries whose it is), and nobody sees or
joins another's; the assistant is told whom it is talking to and offers
them only what their access allows, so settings, updating, page-reading and
undo stay the owner's, and what it asks them to agree to goes to the owner.
Someone who may only look has no assistant. Someone knocking reaches the
owner the way a reminder does, and letting them in says the one step left
in Tailscale: sharing the machine with them.

### More than one host

A workspace can be hosted by several computers at once, each with its own
assistant, model and database, kept the same live. Each copy keeps, beside
its tables, every shared field with the stamp of its latest write
(`_state`): a hybrid logical clock that also names the computer. For each
field the latest stamp wins, so copies that have seen the same stamps hold
the same records whatever order they arrived in; different fields of one
record merge, and a deletion is a field an undo can win over. Keeping two
copies in step is each saying what it has seen from every computer and
getting back what it has not (`internal/peers`, `POST /sync`), every few
seconds over the tailnet, with the machines in `tailnet.peers`. Only the
owner's computers and people with `access: host` may, because a copy can
change anything. Chats, the assistant's questions, actions, devices, files
and the log stay on the computer that made them. Open pages follow what
arrives (`/events`, the page's one connection in design/base/01-connect.js). Content types travel too: each is stamped
like a record (`_schema`), with every part that can change on its own as
its own field: a field's definition, its label, whether it is hidden,
whether it was deleted, and each choice of a pick-list, so choices added on
two computers both stay and, for the rest, the latest wins. A copy makes
its types what the others have through the same changes a person asks for
(`change_field`: add a choice, relabel, hide or show, delete), written to
`schema/`; records that arrive before
their type wait in `_state` and are written when it comes. The system's
own types come with the program and do not travel.

### Working together

- **Two versions at once.** Each stamp of a text field says what it was
  written over. Two edits that were each written without seeing the other
  keep the later everywhere and the other as a `clash`, which the record's
  page offers back (use it, or keep the page's). Nothing is lost silently.
- **Who else is here.** People with a page open, here or on another
  computer that hosts the workspace (carried in each sync exchange), are
  named in the header with where they are, only while someone else is.
- **For someone.** A field pointing at a person (a task's `for`) shows as
  theirs in their colour; one made out for a computer's owner elsewhere
  rings them once, and their assistant knows what is for them.
- **Since you were last here.** Back after half an hour, a person sees what
  others changed meanwhile, each with its Undo, until they say they have.

### Publishing

What the owner explicitly asks to publish (`publish:` tabs and content
types) is readable by anyone on the internet, people and AI services alike,
with no login, at the workspace's own tailnet address, through Tailscale
Funnel. Funnel's listener is Funnel's alone, so the internet only ever
reaches `Server.Public`: published pages and records as read-only HTML
with no controls, conversation or log, and MCP that reads the published
types and nothing else: published to people is published to AI. Everything else is not found, and
nothing is written. The tailnet still gets the whole workspace at the same
address. Publishing is always a question; unpublishing is immediate.

Every address the server answers is one route table
(`internal/server/routes*.go`): each route's handler, who may use it
(people or the owner; one that says nothing is the owner's alone), for a
page action the op the assistant does the same with or why it is a
person's alone, and whether the internet may read it when what it shows is
published. The mux, the access check, the page actions' parity with the
assistant and `Server.Public` all read it.

## 5. How agents use it

- **CLI**: every command supports `--json`. `sameway describe` prints schema and
  component manifests. `sameway component new <name>` scaffolds a compliant folder.
  `sameway chat "..."` talks to the assistant from a terminal.
- **JSON API**: `/api/describe`, `/api/{type}`, `/api/{type}/{id}`, `/api/chat`,
  and every assistant tool at `POST /api/tools/{name}`.
  Errors carry a stable code and per-field messages.
- **MCP**: `sameway mcp` serves the assistant's tools plus `describe` and
  `get_record` over stdio for Claude Code and other hosts, and the server
  serves the same at `/mcp` over HTTP for remote agents.
- **Browser**: every page has landmarks, a single `h1`, skip links,
  `data-component` on each component, and a
  `<link rel="alternate" type="application/json">` to the same data. An agent
  driving a browser navigates by the same structure a screen reader uses.
- **Building UIs**: an agent reads a manifest, renders a component with props,
  and gets AAA output for free. To do something new, it runs the scaffold
  command and fills in one folder.

### 5.1 The chat showcase

The home page is a canvas that fills the screen, and the conversation is one
block on it like any other. The model (any OpenAI-compatible server such as
Ollama, or Claude through the official SDK) gets two kinds of tool: canvas tools
that place and change components, and record tools generated from the
workspace's schema, so a person who asks for a note gets a note on `/t/note`,
not a card. The list lives in one place, the chat service, and `/api/describe`
and `sameway describe` publish it from there. Each canvas tool writes an ordinary `block` record, so the canvas is
content like any other: `sameway block list`, `GET /api/block`, and the page
all show the same thing. The system prompt carries the component catalogue
(every manifest's props schema) and the current canvas, and is rebuilt after
every tool round. Props are validated against the manifest before a block is
saved, and validation errors go back to the model as tool errors, so a model
cannot put inaccessible markup on the page. Conversation turns are `message`
records. Both types live in the workspace schema; the chat disables itself
with an explanation if they are removed.

The page is full-page navigation only: the form posts, the server runs the
tool loop, and redirects to the newest message. No JavaScript is required.

Tabs are canvases. Home is the first, at `/`, and needs no record; every
`canvas` record is one more tab at `/c/<id>`, with blocks of its own (a
block's `canvas` field says which tab it is on, empty for Home). The tab bar
is a list of links, shown only once there is a second tab, and switching is a
page navigation like everything else. A new tab opens on its own chat. The
assistant is told which tab the person is looking at, builds there unless a
call names another, and makes or removes tabs with `create_canvas` and
`remove_canvas`, the latter only through a proposal.

The same tools are a Model Context Protocol server: `sameway mcp` speaks
newline-delimited JSON-RPC on stdin and stdout, which is how Claude Code,
Claude Desktop and the other MCP hosts start one. `tools/list` is the chat
service's list plus `describe` and `get_record`; `tools/call` runs each tool
through the chat service, so an agent in an MCP host meets the same schema
checks and writes to the same activity log as the assistant, and a tool added
to the chat is on MCP the same moment.

Everything the assistant can do is one registry of operations
(`internal/chat/op.go`). Each `Op`, listed beside its code, carries its
tool definition (and when the workspace offers it), its title and traits
(read only, destructive, idempotent, open world), who may have it done
(view, edit or owner), whether a small model here always gets it and the
words that bring it, what the status line says while it runs, what it asks
the person first, and what runs it. The model's tool list, MCP's
`tools/list` with its annotations, `/api/describe/tools` and
`POST /api/tools/{name}` are all read from it, so nothing about a tool is
said twice.

### 5.2 One block, many sizes

A block is placed in a region (`main`, or a full-height `left`/`right` pane)
with a span in columns of twelve. Components that have a `detail` prop also
come in sizes, from a count that only says something needs attention to a
full page with an action on every item, and every size is the same block: the
markup differs, the record does not.

Any block also opens on its own at `/canvas/<id>`, which gives it the middle
of the page and asks it for its largest size. Nothing is moved or copied to
do it, the panes stay where they are, and the block's own control says it is
the current page. That makes one URL per block, which a person can bookmark
and an agent can request directly.

Components compose: a prop may carry `{"component": ..., "props": {...}}`,
validated against that component's own manifest and rendered by its own
template, so a button inside a calendar event is a real button. The host
manifest lists which components may sit there. Nesting is bounded at three
levels; past that the page says so rather than recursing.

## 6. Conventions that keep AI as the main contributor

Enforced by `make check` and CI, not by convention alone.

- **Function and file size**: a non-test Go function over 80 lines fails the
  lint, a file over 400 lines fails and one over 300 is a warning. Files named
  `*_more.go`, `*_extra.go` or `*_helpers.go` fail, since they are split by
  size rather than topic. Long functions and such files from before the rule
  are listed in `tools/check/debt.go` and may only shrink.
- **One concern per file, one package comment per package** saying what lives there.
- **Every command has `--json`**, every error has a stable code and a fix hint.
- **Golden tests**: component template output must match its example HTML.
- **A11y gate**: axe-core and keyboard tests run against every example. AA
  violations fail the build. AAA checks report as warnings with a per-component
  waiver file that must state the reason.
- **No hidden generators**: anything generated is checked in and diffed in CI.
- **`AGENTS.md`** at the root: how to run, check, scaffold, and where things live.

## 7. Accessibility targets

- Contrast 7:1 for text, 4.5:1 for large text (AAA).
- Target size 44 by 44 CSS pixels minimum (2.5.5 AAA).
- Full keyboard operability, visible focus with 3:1 contrast, no keyboard traps.
- `prefers-reduced-motion` and `prefers-contrast` honored in base CSS.
- No timing-dependent interactions in core components.
- Headings and landmarks form a clean outline on every page.
- Plain-language guidance in each manifest for labels and error text.

Where AAA is infeasible for a component, the manifest says so and why.

## 8. What is built

The milestones planned at the start are done or replaced: the skeleton, the
design system (now 50 components, with foundations and arrangements),
export and import, the MCP server over stdio and HTTP, and a starter
workspace. Beyond them: people and access over Tailscale, several hosts
kept in step, publishing, recordings written down on the computer, imports
from other apps, updates of sameway itself, and a benchmark of the
assistant with real models. What each covers and where it is tested is in
[docs/tests/](docs/tests/). Not done: a docs site built with the system
itself, presets cloned from a git URL, auto-export on write.

## 9. Settled decisions

- Node is a dev-only dependency for the axe-core runner. Runtime is pure Go.
- Full-page navigation for forms and chat. No HTML-over-the-wire helper
  until a real need appears.
- Anthropic goes through the official SDK; every other provider goes through
  the OpenAI-compatible chat completions API.
