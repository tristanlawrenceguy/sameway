# Working in this repository as an AI agent

This file is for you. Humans should read README.md and CONTRIBUTING.md.

## Run and verify

```bash
go run ./tools/check          # file-size lint, component folders, tokens
go test ./...                 # includes golden output for every component
go vet ./... && gofmt -l .    # both must be clean
go build -o bin/sameway ./cmd/sameway
./bin/sameway --workspace examples/workspaces/starter describe --json
SAMEWAY_CLAUDE_CODE_MODEL=haiku go test ./internal/server -run ClaudeCode   # one real turn through Claude Code
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
| `internal/relate/` | how one record connects to the others, worked out from the schema | `relate.go` |
| `internal/server/parts.go` | the parts of a page that are off until somebody asks: the keys, and who turned one on | `parts.go` |
| `internal/llm/` | provider-neutral chat + tools; openai.go and anthropic.go | `llm.go` |
| `internal/chat/` | the tool loop, the canvas tools, and the record tools generated from the schema | `chat.go` |
| `internal/mcp/` | the Model Context Protocol server: the chat tools plus reading, over stdio | `server.go` |
| `internal/server/` | HTML pages and JSON API | `server.go`, `canvas.go` |
| `internal/cli/` | the sameway command | `root.go` |
| `internal/update/` | finding, checking and installing a release of sameway itself | `update.go` |
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
| sameway keeps itself current | a release is found and compared, a download is refused unless its sha256 is in `checksums.txt`, a zip or tar.gz is unpacked, the program is swapped with the old one moved aside, auto installs and manual only tells, a build that says `dev` does neither | `internal/update/*_test.go`, `internal/workspace/update_test.go`, `internal/chat/version_test.go` |
| Model uses the canvas tools | add, update, remove, clear, ordering, validation errors, prompt contents, history | `internal/chat/*_test.go` |
| Model makes and changes content records | create, update and find for every schema type, schema errors, activity log, no record tools without content types | `internal/chat/records_test.go` |
| Agent drives the workspace over MCP | handshake, tools/list carries the chat tools plus describe and get_record, tools/call changes land in the store and the activity log, protocol and schema errors are answered | `internal/mcp/server_test.go` |
| A record is shown on the canvas as itself | a record block renders the record's words with edit markers, edits post to the record and return to the canvas, the expanded block is the record at page size, a gone record says so | `internal/server/record_block_test.go` |
| Tabs are canvases | the tab bar appears with a second canvas and marks the current one, a tab shows only its blocks and opens on a chat, what is said on a tab is built there, an expanded block leads back to its tab | `internal/server/tabs_test.go` |
| A page at rest says nothing about what else exists | no connections, no counts, no links, and no field the heading and chips already said; ?show=<key> opens one where the person is and closes again, ui.show +<key> keeps it on, the API and get_record carry the whole graph | `internal/server/related_test.go` |
| Everything leads somewhere | a detail page carries the way back to its listing, activity entries and reply receipts link to what they changed while it exists, activity listings read as sentences | `internal/server/navigation_test.go` |
| Questions are answered where they are met | answers carry the page they were given on and return there, a proposal's own page offers them while pending, clearing the conversation clears its questions, listings show state | `internal/server/proposals_test.go` |
| Model makes and fills tabs | create_canvas, blocks land on the tab the person is on or the one named, the prompt lists the tabs and the current one's blocks, clear_canvas clears one tab, remove_canvas takes its blocks | `internal/chat/canvases_test.go` |
| Person sees and hears a component in a real browser | axe AA and AAA in light and dark, role matches the manifest, 320px reflow, text spacing, 200% text, reduced motion, 3:1 field edges, 44px targets, no colour-only state, visible names in the accessible name, forced colours, errors tied to fields | `tools/a11y-runner/run.mjs` (CI) |
| Person uses a keyboard in a real browser | Tab order, focus ring 2px and 3:1 in light, dark and forced colours, no trap, every control operated by its kind | `tools/a11y-runner/keyboard.mjs` (CI) |
| Person and agent on live pages | keyboard-only flows (the skip link reaches the newest message; a note is edited and saved by keyboard alone), the editor's fields are all design-system components with choices by name and the open form passes axe AAA, role-and-name targeting, describe matches what renders | `tools/a11y-runner/pages.mjs` (CI) |
| Agent reads a page with its scripts run | `look` with scripts and steps drives the Chrome, Edge or Chromium on the machine: a script-built editor is read with its values, Tab is pressed for real from the top, a step that finds nothing lists what the page has, script errors are said; values, forms and only/kind/name without a browser | `internal/server/look_scripts_test.go` (skips with no browser) |
| Everything can be taken back | settings, habit logs, imports (one Undo each), changes through the API and the command line are logged and undoable; the activity log cannot be changed; a failed turn keeps its receipt; a deleted workspace goes to the trash and is restored from Workspaces; a daily copy of the data, seven kept, `sameway restore` keeps what was there; `sameway import --dry-run` | `internal/server/reversible_test.go`, `internal/chat/reversible_test.go`, `internal/workspace/snapshots_test.go`, `internal/cli/import_dry_test.go` |
| First run | with no model the assistant can reach, the conversation says why in plain words and offers what is on this computer (a model server, Claude Code, a saved Claude key), one press each, or what to install and Check again; a record can be added by hand and opens in its editor | `internal/server/connect_test.go` |
| Other people open the workspace over Tailscale | Tailscale says who they are; a person whose email is their login gets in with that person's access (view reads, edit writes), the owner's devices as the owner, anyone else is told they asked and the owner is asked once and notified; each editor has their own chats and turns with the assistant, offered only what their access allows, and a viewer none; the assistant's questions, settings, imports and other workspaces are the owner's; access is set only by the owner's yes (let_in asks, taking it back does not), never through the API or a form; the log names who did it and on what | `internal/server/access_test.go`, `internal/chat/access_test.go`, `internal/chat/people_test.go`, `internal/tailnet/owner_test.go` |
| Several computers host one workspace | every shared write stamps its changed fields; copies that exchange stamps converge whatever the order, merge different fields, agree on the same field, carry deletions and their undo, change nothing on repeats; chats and other local types never leave; old records are seeded; two servers keep a note in step both ways over `/sync`; a new type, fields added on both sides, and a record that arrived before its type all reach the other copy; only the owner and hosts may sync; hosting is asked in the gravest words | `internal/store/state_test.go`, `internal/server/sync_test.go`, `internal/server/schema_sync_test.go`, `internal/chat/access_test.go` |
| Assistant meets what cannot be taken back | running a program, sending to an address, publishing to a device, and the settings that send the conversation or a secret elsewhere, let a program run or open the workspace are asked first, in plain words the code writes, and a Yes runs exactly that; reversible changes are not asked; the assistant cannot write fields Sameway keeps; the person's own press is not asked again | `internal/chat/consent_test.go` |
| Person edits a whole record in place | Edit opens every field of the type in its order, the title, chips and empty fields too, as the right control, each one a copy of the design system's own component (text field, number, day, textarea, select, checkbox) that the page carries; enum choices are offered and shown by their labels; fields marked readonly are never offered and refused by hand; rich text wins over the Markdown behind it; Tab reaches every field | `internal/server/edit_whole_test.go` |
| Assistant sees the page the person is on | look_at_page does what the person did there and reads it with scripts run, falling back to the served page with no browser; MCP keeps its own look | `internal/server/look_chat_test.go` |
| Every page and the site as a whole | every component check on every page, focus hidden on a phone either way up, live regions that can announce, one place per link name, no two headings or controls alike, titles, the same navigation everywhere, forms sent empty say what is wrong | `tools/a11y-runner/site.mjs` (CI) |

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
  The browser suites in tools/a11y-runner enforce all five; a component
  that truly cannot meet one says why in `examples/a11y-waivers.json`.
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
