# Sameway design system

A design system for interfaces that people and AI agents use the same way.
Accessible by construction, and not boring about it: a high-contrast palette
with a point of view, motion that explains change, and a state language that
tells everyone, human or machine, who did what and what is happening now.

Packaged on its own as `@sameway/design` (`package.json`) and embedded in
the `sameway` binary, which serves a living styleguide at `/design`.

## Foundations

| Read | For |
|---|---|
| [foundations/principles.md](foundations/principles.md) | The five rules every decision follows |
| [arrangements/](arrangements/) | Whole pages of thought: which blocks a job wants, where each sits, how wide; the assistant applies one in a call. Every block shows the person's own records (collection, calendar, tracker) with an honest empty state, never example words; one that needs a type the workspace lacks (reading needs book) adds nothing and says how to make it |
| [foundations/color.md](foundations/color.md) | Palette, roles, actor tones, the 7:1 contract |
| [foundations/typography.md](foundations/typography.md) | Scale, measure, numerals |
| [foundations/motion.md](foundations/motion.md) | Durations, curves, view transitions, reduced motion |
| [foundations/elevation.md](foundations/elevation.md) | Surfaces, shadows, lifting, dark surfaces, skeletons, forced colours |
| [foundations/states.md](foundations/states.md) | Provenance, change markers, busy state, activity: text, colour, and attributes |
| [foundations/quiet.md](foundations/quiet.md) | How chrome stays available to everyone while being visible only on demand |
| [foundations/style.md](foundations/style.md) | What the product looks like, in words: air and one blue |
| [foundations/layout.md](foundations/layout.md) | The page as a whole: rows, panes, heading order, and the Layout now line |
| [foundations/navigation.md](foundations/navigation.md) | The sidebar: links in lists not a tree, where you are (page or section), records nested when asked, a cut list that says so |
| [foundations/glance.md](foundations/glance.md) | What a record says beside its title, and how a day is said |

## What is in this folder

- `tokens/tokens.json` is the single source of tokens. `tokens.css` is
  generated from it with `go run ./tools/tokens` and checked in. A Go test
  fails the build if any text pairing drops below 7:1 in either theme.
- `base/` is the foundation, one concern per file, loaded in filename order:
  `*.css` from the reset, feedback, motion, the quiet layer, page shell and
  layout onwards, and `*.js` for page-wide progressive enhancement (editing,
  live regions, following changes, measuring blocks). Every page works
  without the scripts.
- `arrangements/*.json` are whole pages a job wants, applied in one call.
- `brand/` is the icon in every form, drawn by `go run ./tools/icons`.
- `components/<name>/` is one component per folder:
  - `manifest.json`: props (JSON Schema), accessibility contract, keyboard
    map, and how a machine identifies and operates it.
  - `template.html`: Go `html/template` that receives validated props.
  - `style.css`: tokens only, no raw colours (a test checks).
  - `enhance.js`: optional progressive enhancement. The component must work
    without it.
  - `examples/*.html`: rendered examples. They are the standalone spec and
    the golden output the template must reproduce exactly.
  - `README.md`: when to use it, when not to.

## Components

| Component | Purpose |
|---|---|
| heading, text, list, table, card, fields, image | Content |
| link, button, crumbs, tabs, pagination | Actions and ways around |
| chat, disclosure, message, talk, voice | Conversation, and detail behind a summary |
| text-field, textarea, select, checkbox, when-field, lookup, upload | Input |
| alert, status, badge, empty, error-summary, problem, meter | Feedback and state |
| event, since, presence, person, proposal, clash, suggestion | Who did what, who is here, what waits on a person |
| collection, record, mark, move, filters, search, export | Records: lists, one record, ticking, moving, narrowing, finding, taking away |
| calendar, chart, tracker, clock, media | Days, numbers, habits, time, and recordings |

Each folder's README says when to use it; `sameway describe` lists them all.

### Sizes

A component that has a `detail` prop comes in sizes, and the same block is
every one of them:

| detail | What it is | Where it sits |
|---|---|---|
| `glance` | A count, and the words behind it for anyone not reading the number | A strip, a pane, beside a heading |
| `brief` | The few lines that answer "what is next" | A pane |
| `full` | The whole thing | The body of the page |
| `page` | The whole thing with room for actions | The body of the page, or on its own at `/canvas/<id>` |

Sizes are not a visual trick. Each one renders different markup, and each
says the same thing to a screen reader that it says on screen: the glance is
a number with a real sentence beside it, not a bare digit.

### Components inside components

A prop may carry another component, as `{"component": "button", "props":
{...}}`, and the template places it with `{{child .action}}` or `{{children
.actions}}`. The nested component is validated against its own manifest and
rendered by its own template, so it is the real thing, with the same keyboard
behaviour and the same contrast. The host manifest says which components are
allowed where, so nothing can put a form inside a calendar cell. Nesting
stops at three levels deep, with a note on the page rather than a hang.

## Using it without the binary

Load `tokens/tokens.css`, every file in `base/` in filename order, and the
`style.css` of the components you need, then copy the markup from
`examples/`. Add `data-actor`, `data-changed`, and `data-state` attributes as
described in [states.md](foundations/states.md), and `sw-reveal` and
`sw-quiet` as described in [quiet.md](foundations/quiet.md). The base
stylesheet does the rest.

## Rules for a new component

Run `sameway component new <name>` to scaffold. A component is not done until:

1. The manifest has a full props schema, an accessibility contract, a
   keyboard map if anything is focusable, and an example for every enum value.
2. Every example renders from the template with zero diff (`make golden`).
3. `make a11y` reports no AA violations and no keyboard failures.
4. It uses tokens only, reads `--sw-actor` for provenance colour, and keeps
   every interactive target at least 44 by 44 CSS pixels.
5. A prop the server fills in says so in its description: it starts
   "Filled in by the server", or ends "Leave out." or "Filled in by the
   server.". Such props are left out of what a model or agent is shown to
   write (the in-app prompt and `describe <component>`), and out of the
   example it is given; the full manifest still has them.
