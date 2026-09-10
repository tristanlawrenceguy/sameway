# Working in this repository as an AI agent

This file is for you. Humans should read README.md and CONTRIBUTING.md.

## Run and verify

```bash
go run ./tools/check          # file-size lint, component folders, tokens
go test ./...                 # includes golden output for every component
go vet ./... && gofmt -l .    # both must be clean
go build -o bin/sameway ./cmd/sameway
./bin/sameway --workspace examples/workspaces/starter describe --json
```

`make check` runs all of it. CI runs the same plus the Node accessibility
runner in tools/a11y-runner.

## Where things live

| Path | What | Read before editing |
|---|---|---|
| `design/components/<name>/` | one component: manifest, template, css, examples, README | `design/README.md` |
| `design/tokens/tokens.json` | design tokens; regenerate css with `go run ./tools/tokens` | |
| `design/base/*.css` | foundation, one concern per file, loaded in filename order | `design/foundations/` |
| `internal/schema/` | content type files to Go types, validation, JSON Schema | `schema.go` |
| `internal/store/` | SQLite, one table per type | `store.go` |
| `internal/render/` | component registry, props validation, page layout | `registry.go` |
| `internal/llm/` | provider-neutral chat + tools; openai.go and anthropic.go | `llm.go` |
| `internal/chat/` | the tool loop and the canvas tools | `chat.go` |
| `internal/server/` | HTML pages and JSON API | `server.go`, `canvas.go` |
| `internal/cli/` | the sameway command | `root.go` |
| `examples/workspaces/starter/` | what `sameway init` copies | |

## What the tests cover, by the way a component gets used

| Way of using it | Test | Run |
|---|---|---|
| Render from props (server pages, chat tools) | golden output per example, wrong-type rejection | `go test ./internal/render/` |
| Agent reads the manifest and drives a browser | `machine.selector` resolves, ids unique, `aria-describedby` and `label[for]` targets exist, focusable elements have names, keyboard map present, every enum value has an example, CSS uses tokens only | `internal/render/contract_test.go` |
| Model or person supplies hostile props | script, event-handler, `javascript:` and template payloads in every string prop stay inert | `internal/render/hostile_test.go` |
| Workspace adds or overrides a component | `internal/render/override_test.go` | |
| Chrome that is faded but still available | quiet layer stays in the DOM, tab order, and accessibility tree; compact labels keep full accessible names; the log is collapsed but never lost | `internal/server/quiet_test.go` |
| Model lays out and restyles the canvas | span, position, chat as a removable block | `internal/chat/layout_test.go` |
| Person uses the HTML pages | landmarks, skip links, one h1, forms, 422 with linked errors, chat transcript, canvas removal | `internal/server/pages_test.go`, `api_test.go` |
| Agent uses the JSON API | describe completeness, CRUD, stable error shapes, chat builds the canvas | `internal/server/api_test.go` |
| Agent uses the CLI | every command, `--json`, flags in any position, errors that say how to fix | `internal/cli/cli_test.go` |
| Model uses the canvas tools | add, update, remove, clear, ordering, validation errors, prompt contents, history | `internal/chat/*_test.go` |
| Person uses a keyboard in a real browser | Tab order, focus ring, Enter/Space/ArrowDown per component | `tools/a11y-runner/keyboard.mjs` (CI) |
| Person and agent on live pages | keyboard-only flows, axe on every page, role-and-name targeting, describe matches what renders | `tools/a11y-runner/pages.mjs` (CI) |

When you add a component, the contract and enum-coverage tests tell you what
is missing. When you add a way to use the system, add a row here and a test.

## Rules the tooling enforces

- **300 lines per file.** `tools/check` fails above it. Read the whole file
  before editing; split a file rather than growing it.
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
- Do not add a frontend framework or client-side rendering. Pages are
  server-rendered HTML; progressive enhancement only, and only in a
  component's own `enhance.js`.
- Do not add a dependency for something the standard library does.
- New surfaces (MCP, export/import) are generated from the schema and
  manifests, never hand-written per type or component.

## Adding a component

```bash
./bin/sameway --workspace examples/workspaces/starter component new callout
```

Then fill in the manifest (props schema first), the template, the css, the
README, and add examples to the manifest. Run the golden update and
`go run ./tools/check`. Move the folder into `design/components/` when it is
meant to be built in.

## Adding a content type

Add `schema/<name>.yaml` to a workspace and restart. No code change needed.
If a field type is missing, add it in `internal/schema/validate.go`
(coerce + JSONSchema), `internal/store/crud.go` (encode/decode), and
`internal/server/views.go` (control), in that order, with a test.
