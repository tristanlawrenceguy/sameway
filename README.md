# Sameway

Accessible content and components that people and AI agents use the same way.

Sameway is a single binary that serves a structured-content workspace on top of
an accessible design system. Every component and every content type is defined
once, in a plain file, and from that one contract the system generates the
HTML for people, the JSON API and CLI for agents, and the tools a model uses to
build the page you are looking at.

The starter page is a chat. Talk to a model (local through Ollama, or Claude,
or anything OpenAI-compatible) and it adds components to the same page as you
go: a checklist, a table, a card, a form. It can only use components from the
design system, so what it builds is WCAG 2.2 AA clean by construction, and AAA
where the manifest says so.

## Quick start

Requires Go 1.24 or newer. No database, no Node, no config beyond one file.

```bash
go install github.com/tristanlawrenceguy/sameway/cmd/sameway@latest
mkdir my-workspace && cd my-workspace
sameway init
sameway open
```

`sameway open` starts the workspace and opens it in your browser. `sameway
serve` does the same without the browser, for a machine with nobody sitting at
it. If you have cloned this repository rather than installed the binary, the
same thing is a double-click: **open-sameway.cmd** on Windows,
**open-sameway.sh** elsewhere. Both take the same arguments as the command, so
`open-sameway.cmd --workspace "D:\work\my-workspace"` opens a workspace
that lives somewhere else.

Either way it is http://127.0.0.1:8080/. `sameway init` probes for a local model server
(Ollama on 11434, LM Studio on 1234, llama.cpp on 8090 or 8080) and points
the chat at the first model it finds. To change it, or to connect something
else, edit the `llm` section of `workspace.yaml`:

```yaml
llm:
  provider: openai          # Ollama, LM Studio, llama.cpp, OpenRouter, OpenAI
  base_url: http://localhost:11434/v1
  model: llama3.1
  api_key_env: SAMEWAY_LLM_API_KEY   # only needed for hosted providers
```

or for Claude:

```yaml
llm:
  provider: anthropic
  model: claude-opus-5
  api_key_env: ANTHROPIC_API_KEY
```

Keys are read from the environment variable you name. They never go in the
workspace file, because the workspace is meant to be shared.

## What you get

| For people | For agents |
|---|---|
| `/` chat page with a canvas the model edits | `POST /api/chat` with `{"message": "..."}` |
| `/t/note` list and detail pages for every content type | `GET/POST/PUT/DELETE /api/note` |
| Server-rendered HTML, works without JavaScript | `GET /api/describe` for every schema, manifest, tool and route, or `/api/describe/types/note` for one part |
| Skip links, landmarks, one h1, visible focus, 44px targets | `data-component` on every rendered component |
| `sameway note create --set title="Hello"` | `sameway note list --json` |
| Any block opens on its own page at `/canvas/<id>` | One URL per block, at its largest size |
| Ask the assistant for a note and find it on `/t/note` | `sameway mcp`: an MCP host gets the assistant's own tools, plus reading |
| Ask for a second tab and get a second canvas at `/c/<id>` | `POST /api/canvas`, or `create_canvas` over chat and MCP |
| Every page is server-rendered HTML a screen reader can read | `GET /api/look?path=/t/note`: that page as a screen reader gets it, with its structural problems; also `look` over MCP and `sameway look` |
| Add a file on `/t/file`, or attach one to a message: its contents become Markdown on its page, and the assistant reads them | `POST /t/file/upload` (multipart), `GET /files/<id>` for the original; a `files.convert` line in workspace.yaml names a converter per extension, a URL like docling-serve or a command with `{file}` |
| Edit structured text as it is shown: headings, lists and links from a toolbar, the Markdown one button away | The same props route takes `html-<field>` and turns it into Markdown; `POST /api/prose` converts either way |
| Ask for what is due this week and get it on the canvas as a list, a table or cards, with the properties you name beside each; `/t/task?where=done=false&where=due<=+7d&order=due` is the same list as a page | A `collection` block, `GET /api/task?where=…&order=…`, `find_records` with `where`, and `sameway task list --where …` all take the same query: `field=value`, `title~garden`, `due<today`, `notes=` (empty), dates like `today`, `+7d`, `2026-10-01` |
| A task belongs to a project; the project's page lists its tasks by itself, and a task's page links to its project | A field of `type: ref` with `to: project` holds the id; the store refuses an id that is not there; `project=<id>` or `project~garden` in any query |
| Ask for the tasks on a calendar and get this month with each task on its day, as a link, kept current | A `calendar` block with `type: task` (and `where`, `date`, `show`) is filled from records when the page renders; `month` and `today` are filled in too |
| Ask for a due date on notes, or for a new kind of thing such as contacts, and the shape changes at once, for everyone | `add_field` and `add_type` over chat and MCP, `POST /api/types` and `POST /api/types/<type>/fields` for agents: the schema file, the table and every page change while the workspace runs |

## The one contract

**A content type** is one YAML file in `schema/`:

```yaml
name: note
fields:
  title:  { type: string, required: true, maxLength: 200 }
  body:   { type: markdown }
  tags:   { type: list, of: string }
  status: { type: enum, values: [draft, published], default: draft }
  project: { type: ref, to: project }
```

That file gives you the SQLite table, validation, `sameway note ...` commands,
`/api/note`, and `/t/note` pages. Add a file and restart, or ask the assistant
for a new property or a new kind of thing and it changes while you watch.

**A component** is one folder in `design/components/` (or in your workspace's
`components/`), with a manifest that carries the props schema, the
accessibility contract, the keyboard map, the thought behind it (use when, not when, what it sits with), and how a machine finds and operates
it. See [design/README.md](design/README.md).

## Workspace folder

```
my-workspace/
  workspace.yaml   name, server address, model, chat settings
  schema/          content types
  components/      your own components, same layout as built-ins
  content/         every record as Markdown with front matter, kept current
  files/           the originals of files people add, named by record id
  data.db          live SQLite store, ignored by git
```

Commit the folder to share your setup. Clone it on another machine and run
`sameway serve`. Presets are just repositories.

## Commands

```
sameway init [dir]                 create a workspace from the starter preset
sameway serve                      run the web server
sameway describe [--json]          content types, components, routes, model status
sameway check                      validate schema and components
sameway export | import            content/ from the database, or back into it
sameway chat "add a table of ..."  talk to the assistant from the terminal
sameway mcp                        serve the workspace to an MCP client over stdio
sameway component new <name>       scaffold a component folder
sameway <type> list|get|create|update|delete [--json]
```

## Developing

```bash
make check     # gofmt, go vet, repo lint (300-line file cap), tests
make golden    # regenerate component example files from templates
make a11y      # axe-core + keyboard tests over every component example (Node, dev only)
make run       # serve the example starter workspace
make pages     # drive a running server as a person and as an agent (Node, dev only)
```

Tests are organised by the way a component gets used: rendered from props,
read by an agent through its manifest, fed hostile input, used through the
pages, the API, the CLI, the chat tools, and a real keyboard in a real
browser. The table in [AGENTS.md](AGENTS.md) maps each to its test file.

Read [ARCHITECTURE.md](ARCHITECTURE.md) for the design and
[AGENTS.md](AGENTS.md) if you are an AI contributor. MIT licensed.
