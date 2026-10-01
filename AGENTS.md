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
| Model lays out and restyles the canvas | span, position, chat as a removable block; arrange_canvas lays a whole tab out as one change and one Undo, refused whole when it loses a block or skips a heading level; every block write ends with a Layout now line (rows and their fill, holes, tall beside short, heading order and repeats, related blocks apart, late below coming, pane sizes) with an arrange_canvas call that fixes it, the same over MCP and REST (`POST /api/arrange`, `layout` in block answers); see design/foundations/layout.md | `internal/chat/layout_test.go`, `internal/chat/arrange_test.go`, `internal/mcp/arrange_test.go`, `internal/server/api_arrange_test.go` |
| Layout is measured in the person's browser | 25-measure.js sends each block's size, inner scroll, clipping and overflow past the screen as numbers and ids only, from pages the owner or an editor opens (never published, view-only, agents, save-data or driven browsers); POST /canvas/measure refuses anything else and caps sizes; readings are kept per block and kind of screen and go stale when the block changes or after seven days; Layout now uses measured heights, says measured or estimated, and names an inner scroll, a pane too short, a cut or a sideways overflow with the arrange_canvas call that fixes it; /api/look gives them as measured; the runner checks seeded tabs at desktop and phone widths with the same script | `internal/chat/measured_test.go`, `internal/server/measure_test.go`, `tools/a11y-runner/measure.mjs` (site.mjs) |
| Person uses the HTML pages | landmarks, skip links, one h1, forms, 422 with linked errors, chat transcript, canvas removal | `internal/server/pages_test.go`, `api_test.go` |
| Agent uses the JSON API | describe is a short index that says how to build (POST /api/block first) with the rest by part or ?full=1, describe completeness, CRUD, stable error shapes, chat builds the canvas | `internal/server/api_test.go`, `describe_index_test.go` |
| Agent does what a page does | every POST a form uses answers `Accept: application/json` with the outcome a person reads, never a page; a JSON body is the form; a chat turn carries a file by id | `internal/server/agents_test.go` |
| One name, one sentence, one write | a record is named by chat.Name and a change said by chat.Sentence on every surface, old stored words included; records are written through chat.WriteAs from every way in, and a new direct store write fails until it says why | `internal/server/one_voice_test.go`, `writes_test.go` |
| The assistant can do what a page does | every page action names the tool that does the same, or says why it is a person's ("a person's: …"); there is no third answer, and an owner-only page action has only owner-only tools | `internal/server/tools_parity_test.go` |
| Every route says who may use it | people (look with view, change with edit) or owner; a route not listed is the owner's alone; the owner's own kinds of record (chats, questions, the log) are refused to visitors on every surface, exports included | `internal/server/access_routes_test.go` |
| A save from an out-of-date copy loses nobody's work | a page sends what it showed (version and field fingerprints) and fields the person left keep what others changed; agents send If-Match or update_record's version and an old one is refused with the record as it is | `internal/server/versions_test.go` |
| An agent follows changes, retries safely and reads less | GET /api/changes?since=&wait= with each reading what they may; Idempotency-Key on any change, the API or a page's form; ?page= and ?fields= on reads | `changes_test.go`, `once_test.go`, `api_read_test.go` |
| An agent is who its key says | a key from `sameway agent add` names the agent in the log whatever it calls itself, limits it to view, edit or owner on the API, pages and MCP, and stops working when taken away, until that is undone | `internal/server/agent_keys_test.go` |
| A part is offered when use shows a reason | the owner asking for the same part on a kind of page three times in a week is offered it for good, with the reason in one sentence; yes is logged and undoable, no is not asked again, and people let in are not watched | `internal/server/learn_test.go` |
| Agent uses the CLI | every command, `--json`, flags in any position, errors that say how to fix | `internal/cli/cli_test.go` |
| sameway keeps itself current | a release is found and compared, a download is refused unless its sha256 is in `checksums.txt`, a zip or tar.gz is unpacked, the program is swapped with the old one moved aside, auto installs and manual only tells, a build that says `dev` does neither | `internal/update/*_test.go`, `internal/workspace/update_test.go`, `internal/chat/version_test.go` |
| Model uses the canvas tools | add, update, remove, clear, ordering, validation errors, prompt contents, history | `internal/chat/*_test.go` |
| Model makes and changes content records | create, update and find for every schema type, schema errors, activity log, no record tools without content types | `internal/chat/records_test.go` |
| Agent drives the workspace over MCP | handshake, tools/list carries the chat tools plus describe and get_record, tools/call changes land in the store and the activity log, protocol and schema errors are answered; over HTTP, at the computer everything, over the tailnet no token and what the person may (owner everything, edit their assistant's tools, view reading), from elsewhere with only the token reading | `internal/mcp/server_test.go`, `internal/mcp/remote_test.go` |
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
| Another process changes the types | a type or field made by `sameway mcp` (Claude Code's tool process), the CLI or a hand edit of schema/ is served by the running server at once, with its table, and open pages are told; the other way round too; a file that does not read is left until it does; one taken away keeps its records; schema files are written whole | `internal/app/reload_test.go`, `internal/server/schema_reload_test.go` |
| Several computers host one workspace | every shared write stamps its changed fields; copies that exchange stamps converge whatever the order, merge different fields, agree on the same field, carry deletions and their undo, change nothing on repeats; chats and other local types never leave; old records are seeded; two servers keep a note in step both ways over `/sync`; a new type, fields added on both sides, choices added on both sides, labels, hiding and deletions, and a record that arrived before its type all reach the other copy; a pick-list gains a choice, fields and choices are relabelled, a field is hidden (off pages and tools, kept, back when shown), and deleting is asked with hiding offered first; only the owner and hosts may sync; hosting is asked in the gravest words | `internal/store/state_test.go`, `internal/server/sync_test.go`, `internal/server/schema_sync_test.go`, `internal/server/reshape_test.go`, `internal/chat/access_test.go` |
| People work together | text rewritten on two computers at once keeps the replaced version as one clash on both, offered on the record's page (use or keep); an edit made over the other is no clash; who else is here is said with where, never oneself, and people on other computers too; a task made out for this computer's owner elsewhere rings them once and shows in their colour; back after a while, others' changes wait with their Undo until Got it | `internal/store/state_test.go`, `internal/server/together_test.go` |
| The internet reads what is published | with nothing published the address says so; a published type's pages read with no controls, other types, the chat, the log, the API, events and every write are not found or refused and nothing changes; a published tab reads without the conversation and the front page lists it; readers leave no presence or return marks; published pages carry no form that sends, no control bar, no chat, no log and no link to anything unpublished (they are taken out, not hidden, so people and AI read the same page); a picture is served only when a published page uses it; AI services read the same over MCP (at /mcp in any case), only published types, which are the only ones its tools name, with search and fetch across them as ChatGPT's connectors and deep research ask, citing each record's page, no search, no writes; publishing is asked, unpublishing is not | `internal/server/publish_test.go`, `internal/chat/consent_test.go` |
| Assistant meets what cannot be taken back | running a program, sending to an address, publishing to a device, and the settings that send the conversation or a secret elsewhere, let a program run or open the workspace are asked first, in plain words the code writes, and a Yes runs exactly that; reversible changes are not asked; the assistant cannot write fields Sameway keeps; the person's own press is not asked again | `internal/chat/consent_test.go` |
| Person edits a whole record in place | Edit opens every field of the type in its order, the title, chips and empty fields too, as the right control, each one a copy of the design system's own component (text field, number, day, textarea, select, checkbox) that the page carries; enum choices are offered and shown by their labels; fields marked readonly are never offered and refused by hand; rich text wins over the Markdown behind it; Tab reaches every field | `internal/server/edit_whole_test.go` |
| A block set up wrong says so | a list, chart, calendar or tracker asked for a type, field, date field, condition or tag the workspace does not have says it cannot be shown, one way everywhere (the problem component), with what is wrong and what there is instead under What is wrong, never an empty month or "Nothing tracked yet" | `internal/server/problem_test.go` |
| A block is checked when written | add_component, update_component, propose_change, add_arrangement, the same over MCP, and POST or PATCH /api/block resolve a block as its page will before writing it: one that could only say it is set up wrong is refused (a tool error, 422 cannot_show over the API) with the page's own reason and fix, and nothing is written; one that can be shown says what it shows ("3 tasks, not done, by due"); a look lists each block on a page that cannot be shown under problems; a block stored before still renders its problem | `internal/server/check_test.go`, `internal/mcp/check_test.go` |
| Props that do not fit are said for whoever fixes them | a tool, MCP or API error names each wrong prop, where a block field such as tone belongs (beside props), the prop likely meant (button takes label, not text), what an enum allows, what is missing, and every prop the component takes (chat.PropsTrouble); a person on the canvas or an edit form reads plain words by the prop's name, never the schema's | `internal/chat/model_errors_test.go`, `internal/render/props_error_test.go` |
| A person narrows and sorts a list where it is | a collection offers a sort and a pick-list, a yes-or-no and a date field from its type, only what the schema has and not what where fixes; a choice only adds to where, and the address cannot add what the form does not offer; the choices are named after the block, so two lists keep their own, the form keeps the page's other fields and Reset takes only its own; the list page link carries them; offered on its own page and on a canvas when more than a few match, controls: false or true to say otherwise | `internal/server/collection_choices_test.go` |
| A list is narrowed one way everywhere | the filters component: links for one choice among a few (a search's kinds, counted, the one shown aria-current), a form of labelled selects with Apply for several (a collection's sort and fields), then how many match and what is shown in words with Reset; both plain GETs that keep the page's other fields; the activity log by who, what and when together (only what the log has, a fingerprint for another person, pages and Undo keeping the choices); a calendar of everything by kind, counted for the month, only with two kinds or more, per block, kept by the months and days either side, never widening a calendar of one type | `internal/server/search_types_test.go`, `internal/server/collection_choices_test.go`, `internal/server/activity_filters_test.go`, `internal/server/calendar_kinds_test.go`, `internal/render/golden_test.go` |
| An action comes back where it was taken | every form in a canvas block says its block with no script, a closer place (a card, a row, a board's Move button) sent after it wins, only an id is taken, the page returns there with the outcome, and a control returned to hears it as its description; a move to where a record already is saves nothing and says so; a saved edit of one field says what it became and was, a ref by its title, long text only that it changed | `internal/server/back_test.go`, `internal/server/board_test.go`, `tools/a11y-runner/move.mjs` |
| Something happens again | a repeat is written in words and every common way is read (weekdays, every 2 weeks, the 31st, 29 Feb, until), kept as an RRULE, said back as read and refused with what it must be; a task ticked done or a reminder dismissed moves to its next time on its schedule, however it was finished, in one undoable change; the 31st keeps to the 31st after February; five more minutes keeps an alarm's own time; an older workspace gets the field; the calendar shows it on each day it falls in the month shown, each saying it repeats once, only the one due now with its tick, never past the month or its end | `internal/when/repeat_test.go`, `internal/when/repeat_on_test.go`, `internal/server/repeat_test.go`, `internal/server/calendar_repeat_test.go` |
| A file of any size is kept | an upload streams to disk as it arrives, up to 4 GB, never held whole; past 64 MB a recording is kept to be written down and a document is kept as it is, saying so; `sameway add <file>` copies files on this computer in with no browser, read as an upload is, the original left where it was, said or as JSON, a folder or a missing file saying what to do | `internal/server/keep_test.go`, `internal/cli/add_test.go` |
| Calendars, code and subtitles are read | an .ics from Google, Outlook or Apple is read into its events (all day by the day, times in their zone and Outlook's zone names, repeats as rules, escapes and folded lines undone, cancelled ones and single changes left out), reads as its events and leads to bringing them in as events on the calendar, again adding nothing twice; source and configuration in some forty languages read as one code block, a .env never; subtitles read as a transcript and, beside a video or recording of the same name with no words, become its words and captions | `internal/convert/ics_test.go`, `internal/server/calendar_import_test.go` |
| What comes in goes out again | a list page offers what fits its type, with its own query: a spreadsheet (CSV with a byte-order mark, Excel with its header held) for any, contacts (vCard) for people, a calendar (iCalendar, which a calendar app can also subscribe to) for anything with a date; values go out as the import reads them back, so tasks, events and people come back as they went, a calendar adding nothing twice; the system's own records never go out, and a format that does not fit says which do; a recording's words go out as subtitles and text; `sameway <type> list --format` does the same; every way out is the export component, after what it takes out, each link saying the kind of file, its format and its size where making it is cheap; a spreadsheet cell that looks like a formula goes out inert and comes back as written; a calendar's all-day end is the day after (and read back as the last day), a timed repeat's UNTIL a time; a name beyond ASCII has filename*; the internet takes away published lists and records only, without hidden fields; everything with a day is one calendar; a collection's and a calendar's own page offer what they show, with the choices made on them | `internal/server/export_test.go`, `internal/server/export_files_test.go`, `internal/server/export_all_test.go`, `internal/cli/export_test.go` |
| A record or a whole workspace goes out | a record's page offers it as Markdown (title, facts, words), and as a calendar entry or a contact when it is one, a web page that stands alone (its page as a reader has it, styles and pictures inside, no controls or scripts), Word (title, language, heading styles, real lists, links, a repeating table header) and a tagged PDF with an outline printed by the host's browser, or why not; the whole workspace goes out in one zip, its owner's alone and offered on the workspaces page: records as the content folder's Markdown, files as added, a spreadsheet of each kind, which `sameway import` brings back into another workspace; `sameway export --zip` does the same | `internal/server/export_docs_test.go`, `internal/server/everything_test.go`, `internal/cli/everything_test.go` |
| A recording plays with its words | an audio file is kind audio and a video kind video (a WebM judged by what is in it), each served as its own type on every system, and its page plays it with the media component, never by itself; its transcript (<id>.vtt beside the original) is under it, each line with who spoke, leading into the recording (#t=); with no transcript the page says so; a video carries its transcript as captions on by default, made from the text so a correction shows there too, and served with it when it is published; WebVTT is read into who said what and when | `internal/server/audio_test.go`, `internal/server/video_test.go`, `internal/convert/vtt_test.go` |
| A recording is written down on this computer | the owner alone is offered speech-to-text, saying what it downloads and from where, and others are told who can; each file fetched is kept only with its pinned fingerprint, the engine unpacked with nothing outside its folder, nothing fetched twice; every system sameway ships for has a pinned engine; once here the page writes each recording down by itself from the 16 kHz WAV its script makes (any WAV at any rate is brought to it, with no script only a WAV), the transcript is the file's text a line per stretch with its time, read back at its times after it is corrected, and kept as WebVTT; the sound sent is not kept | `internal/speech/speech_test.go`, `internal/server/transcribe_test.go`, `internal/convert/vtt_test.go` |
| A long recording is written down in parts | an MP4's sound is found by its sample table (or its fragments) and copied out as ADTS, a WebM's by its tracks, through clusters of unknown size, as Ogg Opus with true checksums, an MP3 cut at its frames, each in chunks at their times with the picture left behind, anything else read whole; a long WAV is cut into 16 kHz chunks on the host a second at a time; the host says how each recording is read (here for a WAV, chunks, or whole), each part is written down in turn at its place with the page told how far it has come, the parts come together at their times, and the copied-out sound goes afterwards | `internal/soundtrack/soundtrack_test.go`, `internal/speech/wavsplit_test.go`, `internal/server/parts_test.go` |
| A recording is written down with no page open | a running workspace (serve, open) writes recordings down itself: one that arrives by any way, and any already there without words, found by a sweep a minute; a WAV straight away, anything else read in the Chrome or Edge on the host as a page would read it; the page does not start what the host will do, and does it itself where the host cannot; tests never start it | `internal/server/hostwrite_test.go` |
| A message can be said instead of typed | every upload and the chat's Attach carry Record, hidden until the page can record; the chat offers Dictate only once speech-to-text is on this computer; /dictate writes the sound down here and answers its words, keeping nothing, refuses sound it cannot read, and without speech-to-text says who can get it ; voice mode (the talk component) is offered beside Dictate only then | `internal/server/dictate_test.go` |
| The assistant sees pictures | a picture attached to a message, or named in one by its page or id, goes to a model that can see: as an image part to OpenAI-compatible servers (Ollama, LM Studio), an image block to Anthropic, and to Claude Code as a file it may read and nothing else, gone with the turn (one real turn with SAMEWAY_CLAUDE_CODE_MODEL); small pictures as they are, large ones no longer than 1568 pixels; only the three newest each turn; a model that cannot see answers again, told a picture was there; a picture with no description leads to asking the assistant for a draft | `internal/llm/images_test.go`, `internal/llm/images_real_test.go`, `internal/server/pictures_test.go` |
| Assistant sees the page the person is on | look_at_page does what the person did there and reads it with scripts run, falling back to the served page with no browser; MCP keeps its own look | `internal/server/look_chat_test.go` |
| Two records alike are told apart | where titles repeat on a list, a block, a board, search, the log or a calendar, each repeated one's link, box, Move, Undo and choice carry hidden words that differ (due Fri 25 Sep, when added, the start of its id), and a title of its own is left plain; a block's Expand and Remove are named after its label | `internal/server/names_apart_test.go`, `internal/server/apart_test.go` |
| A browser agent on live pages | from the accessibility tree with records alike seeded: every control named, none alike within a landmark, one place per link name, visible label inside the name and not twice, no glyphs, look agrees; find overdue, tick, undo, add a note and search by role and name alone, failing on a strict-mode match of two | `tools/a11y-runner/agent.mjs` (CI) |
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

## What agents need: the checklist

Beside the rules for people, the 22 checks from the agent accessibility
audit (2026-09-28): a browser agent finds controls by role and name in the
accessibility tree, a screenshot agent sees only what is drawn, a tool agent
reads MCP and the API. Where each is checked: `agent` is
`tools/a11y-runner/agent.mjs`, `site` is `site.mjs`, `look` is `/api/look`'s
problems, `go` a Go test; `known` is reported by agent.mjs but fails only
with `--strict` until its fix lands; `review` and `not yet` are yours to hold.

| # | Check | Where |
|---|---|---|
| | **Perceive** | |
| 1 | Every control named, in the browser's own tree | agent, look |
| 2 | No two controls of one role and name within a landmark (records alike are told apart by their due day, when added, or id); links of one name go to one place | agent, look, site, go `names_apart_test.go` |
| 3 | What a control shows is in its name; hidden words add to it, never repeat it | agent, run.mjs |
| 4 | No operable control below 0.35 opacity at rest | agent (known) |
| 5 | The document scrolls as one page, so a full-page screenshot has it all | not yet |
| 6 | No meaning only in a canvas, a hover or a drag | review |
| | **Understand** | |
| 7 | One ariaSnapshot golden per page type | not yet |
| 8 | No glyphs in names; no counts run into words ("This week 1 28 Sep") | agent (glyphs); counts not yet |
| 9 | A manifest's `machine` says how to find it by role and name, and that holds | go `contract_test.go` (selector); role-and-name not yet |
| 10 | look says what the browser's tree says | agent (look on the same pages); not compared yet |
| | **Operate** | |
| 11 | Everything works without scripts | go `pages_test.go`, review |
| 12 | After an in-place change the page equals a reload | agent (known) |
| 13 | An outcome stays, in a status or alert, names the record, offers Undo | agent (tick, undo), go `back_test.go`, `reversible_test.go` |
| 14 | Nothing is written until a deliberate press; a control that writes says so | review |
| | **Tools** | |
| 15 | Every MCP tool titled and annotated; read-only ones proved so by a store diff | `internal/mcp/annotations.go`; `TestEveryToolSaysWhatItIs`, `TestReadOnlyToolsChangeNothing` (the workspace on disk hashed before and after) |
| 16 | Every error is `isError` with the next step and the valid choices | go `internal/mcp/*_test.go` (partly) |
| 17 | List rows can be told apart and are paged | pages: agent, go; tool results not yet |
| 18 | `describe` has a small index under a budget | go `describe_index_test.go` (16 KB, block routes first), `internal/mcp/describe_size_test.go` (nothing but full past a client's limit), `prompt_budget_test.go` (the in-app prompt at 113 KB) |
| | **Trust** | |
| 19 | MCP and API writes are logged with the agent's name, from its key when it has one | go `agent_keys_test.go`, `writes_test.go` |
| 20 | Record text reaches outside agents marked as content, with who wrote it | `internal/chat/provenance.go` (written_by and untrusted on a record, a list, search, the changes feed, the tools and public MCP); go `untrusted_test.go`, `internal/mcp/untrusted_test.go` |
| 21 | Irreversible actions ask first; outward tools say openWorldHint | go `consent_test.go` (asking), `internal/mcp/annotations.go` (run_action and update_sameway say openWorldHint) |
| 22 | Writes are rate-limited per token | `internal/chat/pace.go` (60 changes a minute per key; reads and the owner unpaced); go `pace_test.go`, `agent_pace_test.go`, `internal/mcp/pace_test.go` |

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
