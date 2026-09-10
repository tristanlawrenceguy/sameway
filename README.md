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
go install github.com/sameway-dev/sameway/cmd/sameway@latest
mkdir my-workspace && cd my-workspace
sameway init
sameway serve
```

Open http://127.0.0.1:8080/. To connect a model, edit the `llm` section of
`workspace.yaml`:

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
| `/t/note` list, detail, new, edit pages for every content type | `GET/POST/PUT/DELETE /api/note` |
| Server-rendered HTML, works without JavaScript | `GET /api/describe` for every schema, manifest, and route |
| Skip links, landmarks, one h1, visible focus, 44px targets | `data-component` on every rendered component |
| `sameway note create --set title="Hello"` | `sameway note list --json` |

## The one contract

**A content type** is one YAML file in `schema/`:

```yaml
name: note
fields:
  title:  { type: string, required: true, maxLength: 200 }
  body:   { type: markdown }
  tags:   { type: list, of: string }
  status: { type: enum, values: [draft, published], default: draft }
```

That file gives you the SQLite table, validation, `sameway note ...` commands,
`/api/note`, and `/t/note` pages. Add a file, restart, done.

**A component** is one folder in `design/components/` (or in your workspace's
`components/`), with a manifest that carries the props schema, the
accessibility contract, the keyboard map, and how a machine finds and operates
it. See [design/README.md](design/README.md).

## Workspace folder

```
my-workspace/
  workspace.yaml   name, server address, model, chat settings
  schema/          content types
  components/      your own components, same layout as built-ins
  content/         exported records (milestone 3)
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
sameway chat "add a table of ..."  talk to the assistant from the terminal
sameway component new <name>       scaffold a component folder
sameway <type> list|get|create|update|delete [--json]
```

## Developing

```bash
make check     # gofmt, go vet, repo lint (300-line file cap), tests
make golden    # regenerate component example files from templates
make a11y      # axe-core over every component example (needs Node, dev only)
make run       # serve the example starter workspace
```

Read [ARCHITECTURE.md](ARCHITECTURE.md) for the design and
[AGENTS.md](AGENTS.md) if you are an AI contributor. MIT licensed.
