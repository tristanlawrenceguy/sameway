# Architecture

Name: **Sameway**. License: MIT. Module path: `github.com/sameway-dev/sameway`
(the `sameway-dev` GitHub org was free when this was written; change the
module path in `go.mod` and the imports if the repo lands elsewhere).

Status: milestone 1 is scaffolded and verified (see section 8). The starter
page is a chat whose model edits the same page through tools; see section 5.

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
| Templates | Go `html/template` with a typed props struct per component | Stdlib, auto-escaping, no code generator. Props are validated against the component manifest in tests. |
| Storage | SQLite (modernc.org/sqlite) as live store, Markdown/JSON files as portable form | Zero setup. Files make the workspace git-friendly and AI-readable. `export`/`import` keep them in sync; auto-export is a config flag. |
| CMS scope | Structured content types only, no page builder | Keep the core small. Rendering is done by components; if a view does not exist, create a component. |
| Design system | Standalone package (tokens, CSS, HTML patterns, manifests) | Usable in any stack. The Go binary is one consumer. |
| Accessibility bar | WCAG 2.2 AAA where feasible, AA as hard gate in CI | Every component ships with automated and keyboard tests. |
| Users | Single user, local first | Auth is an optional module added later, not baked into day one. |
| Sharing | Workspace = git-friendly folder | Push a repo or sync a folder. Presets are just repos. |

## 2. The one contract

Everything an agent or a human needs is derived from two kinds of declarative files.

### 2.1 Component manifest

Each component is one folder. The manifest is the source of truth; the template
must render exactly what the examples show (enforced by golden tests).

```
design/components/button/
  manifest.json      # name, description, props schema, a11y contract, machine notes
  template.html      # html/template, receives typed props
  style.css          # scoped by class prefix, uses tokens only
  enhance.js         # optional, progressive enhancement, no framework
  examples/          # plain HTML files, the standalone spec and golden output
  README.md          # human docs, generated sections from manifest
  test/              # axe + keyboard + golden tests
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
- MCP tools: `note_create`, `note_get`, `note_list`, `note_update`, `note_delete`
- JSON API: `/api/note`, `/api/note/{id}`
- HTML views: `/note`, `/note/{id}`, `/note/new`, built from the named components
- `describe` output so an agent can learn the type without reading rows

Nothing is hand-written per surface. Adding a surface means adding a generator.

## 3. Repository layout

```
/
  README.md  LICENSE  ARCHITECTURE.md  CONTRIBUTING.md  AGENTS.md
  design/                     # standalone design system, publishable
    tokens/tokens.json        # single source; tokens.css generated
    base/                     # reset, typography, focus ring, reduced motion, print
    components/<name>/        # see 2.1
    package.json              # publishes css + manifests + examples
  cmd/sameway/main.go           # entry point, wires modules
  internal/
    workspace/                # folder layout, init, export, import, presets
    schema/                   # load + validate content type files
    store/                    # SQLite, migrations, generic CRUD keyed by schema
    render/                   # template loading, component registry, props validation
    server/                   # HTTP: HTML views, JSON API, describe endpoint
    mcp/                      # MCP server (stdio + HTTP) generated from schema + manifests
    cli/                      # commands generated from schema, plus scaffold/check/serve
    a11y/                     # shared helpers: landmarks, headings, live regions
  tools/
    check/                    # file-size lint, manifest validator, golden test runner
    a11y-runner/              # Node dev-only: playwright + axe-core against examples
  examples/workspaces/        # notes, blog, inventory: each is a shareable preset
```

## 4. Workspace folder

```
my-workspace/
  workspace.yaml     # name, theme tokens override, auto-export flag, enabled components
  schema/            # content types
  components/        # local components or overrides, same folder contract as design/
  content/           # exported records: content/note/<id>.md with frontmatter
  data.db            # live SQLite store, gitignored, rebuilt by `import`
```

`sameway init` creates one. `sameway init --from <git url>` clones a preset.
Multi-device is git or any folder sync. Later, an optional auth module can sit in
front of a shared instance without changing this layout.

## 5. How agents use it

- **CLI**: every command supports `--json`. `sameway describe` prints schema and
  component manifests. `sameway component new <name>` scaffolds a compliant folder.
  `sameway chat "..."` talks to the assistant from a terminal.
- **JSON API**: `/api/describe`, `/api/{type}`, `/api/{type}/{id}`, `/api/chat`.
  Errors carry a stable code and per-field messages.
- **MCP** (milestone 4): `sameway mcp` exposes tools for every content type and
  resources for the schema, manifests, and design tokens. Works over stdio for
  Claude Code and over HTTP for remote agents.
- **Browser**: every page has landmarks, a single `h1`, skip links,
  `data-component` on each component, and a
  `<link rel="alternate" type="application/json">` to the same data. An agent
  driving a browser navigates by the same structure a screen reader uses.
- **Building UIs**: an agent reads a manifest, renders a component with props,
  and gets AAA output for free. To do something new, it runs the scaffold
  command and fills in one folder.

### 5.1 The chat showcase

The home page is a conversation next to a canvas. The model (any
OpenAI-compatible server such as Ollama, or Claude through the official SDK)
gets four tools: `add_component`, `update_component`, `remove_component`,
`clear_canvas`. Each one writes an ordinary `block` record, so the canvas is
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

## 6. Conventions that keep AI as the main contributor

Enforced by `make check` and CI, not by convention alone.

- **File size cap**: 300 lines per source file. The lint fails above it. Agents
  read whole files before editing, so files stay small enough to read.
- **One concern per file, one README per package** describing what lives there.
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

## 8. Milestones

1. **Skeleton** (done): repo layout, `init`, `serve`, `describe`, `check`,
   content types with generated CLI, JSON API, and HTML views; thirteen
   components with manifests, golden tests, and the a11y runner; the chat
   showcase with OpenAI-compatible and Anthropic providers; repo lint with
   the 300-line cap; CI workflow.
2. **Design system depth**: radio group, fieldset, details/summary, dialog
   (native), pagination, breadcrumb; keyboard tests in the a11y runner;
   AAA waiver files; publish `@sameway/design`.
3. **Export/import**: content as Markdown with front matter in `content/`,
   `sameway export`, `sameway import`, optional auto-export on write.
4. **MCP server** from the same schema and manifests, stdio and HTTP.
5. **Presets and docs**: three example workspaces, docs site built with the
   system itself, `init --from <git url>`.
6. **Later**: optional auth module, streaming replies if a real need appears,
   theme editor, Markdown rendering for `markdown` fields.

## 9. Settled decisions

- Node is a dev-only dependency for the axe-core runner. Runtime is pure Go.
- Full-page navigation for forms and chat. No HTML-over-the-wire helper
  until a real need appears.
- Anthropic goes through the official SDK; every other provider goes through
  the OpenAI-compatible chat completions API.
