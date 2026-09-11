# Sameway design system

A design system for interfaces that people and AI agents use the same way.
Accessible by construction, and not boring about it: a high-contrast palette
with a point of view, motion that explains change, and a state language that
tells everyone, human or machine, who did what and what is happening now.

Published on its own as `@sameway/design` and embedded in the `sameway`
binary, which serves a living styleguide at `/design`.

## Foundations

| Read | For |
|---|---|
| [foundations/principles.md](foundations/principles.md) | The five rules every decision follows |
| [foundations/color.md](foundations/color.md) | Palette, roles, actor tones, the 7:1 contract |
| [foundations/typography.md](foundations/typography.md) | Scale, measure, numerals |
| [foundations/motion.md](foundations/motion.md) | Durations, curves, view transitions, reduced motion |
| [foundations/states.md](foundations/states.md) | Provenance, change markers, busy state, activity: text, colour, and attributes |
| [foundations/quiet.md](foundations/quiet.md) | How chrome stays available to everyone while being visible only on demand |

## What is in this folder

- `tokens/tokens.json` is the single source of tokens. `tokens.css` is
  generated from it with `go run ./tools/tokens` and checked in. A Go test
  fails the build if any text pairing drops below 7:1 in either theme.
- `base/*.css` is the foundation, one concern per file, concatenated in
  filename order: reset, feedback, motion, the quiet layer, page shell,
  layout, styleguide.
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
| heading, text, list, table, card | Content |
| link, button | Actions |
| chat, disclosure | Containers: a conversation region, and detail hidden behind a summary |
| text-field, textarea, select, checkbox | Input |
| alert, status, badge | Feedback and state |
| message, event | Conversation and activity |
| calendar, datepicker | Dates: a month at four sizes, and one day picked |

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
