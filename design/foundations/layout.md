# Layout: the page as a whole

The owner, 2026-09-30: "Humans don't need more control if the AI does it
well. AI just adds though and doesn't think about how it fits in and if
other pieces should move or change shape."

So a person gets no move or resize handles. The one building (the
assistant, or an agent over MCP or the API) lays out the whole tab, and
Sameway tells it after every write how the page now reads.

## What reads well, and why

| Rule | Why | Source |
|---|---|---|
| Most important first; what is late above what is coming | People look at the top and left first and scan down; the first rows get the attention | [NN/g, F-shaped pattern](https://www.nngroup.com/articles/f-shaped-pattern-reading-web-content/) |
| Headings mark the sections, and no two say the same | The best way people scan is heading to heading (layer cake); a heading repeated says nothing | [NN/g, layer-cake pattern](https://www.nngroup.com/articles/layer-cake-pattern-scanning/), [WCAG 2.4.6](https://www.w3.org/WAI/WCAG22/Understanding/headings-and-labels.html) |
| Heading levels never skip down (h1, then h2, then h3) | Screen reader users move by level; a skipped level hides where a section begins | [WCAG H42](https://www.w3.org/TR/WCAG20-TECHS/H42.html), [Vispero on skipped levels](https://vispero.com/resources/heading-off-confusion-when-do-headings-fail-wcag/) |
| Related things sit together | Things near each other are seen as one group, more strongly than by look | [NN/g, proximity](https://www.nngroup.com/articles/gestalt-proximity/) |
| Rows of twelve are filled; a wide main column with a narrower one beside it | Holes read as missing content; two-thirds and one-third is the common, readable split | [GOV.UK layout](https://design-system.service.gov.uk/styles/layout/), [Material responsive grid](https://m2.material.io/design/layout/responsive-layout-grid.html) |
| Order on the page is order in the source | The grid places blocks by position, so what is seen first is heard first, and one column on a phone keeps the order | [WCAG 1.3.2](https://www.w3.org/WAI/WCAG22/Understanding/meaningful-sequence.html), [Source order matters](https://adrianroselli.com/2015/09/source-order-matters.html), [WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html) |
| Tall things get a row or a pane of their own; glance and brief sizes go in panes | A short card beside a tall calendar leaves a hole under the card | this design system's grid (`design/base/07-layout.css`) |

## How it is built

- `arrange_canvas` (MCP too, `POST /api/arrange` over REST) takes every
  block on a tab in reading order with its span, region, frame and size, and
  applies it as one change: one entry in the log, one Undo. It is refused
  whole when a block is missing or listed twice, a value is out of range,
  or a heading would skip a level.
- Every block write (add, update, remove, arrange, an arrangement) ends
  with a "Layout now" line: the rows and how full each is, the panes, what
  to look at (holes, tall beside short, heading order, two headings alike,
  a section heading narrower than 12, related blocks apart, what is late
  below what is not, pane sizes in the main region and full sizes in a
  pane, positions shared), and an `arrange_canvas` call that fixes the
  order and widths. It is information, never a refusal: the model decides.
- A skipped heading level is the one fault refused at write, with its fix.
  It is a clear accessibility break, and which fix is meant (a lower level,
  or a section heading above) is the writer's to say; changing the level
  silently would give a different outline from the one intended.

## Measured, not guessed

How tall a block is depends on things only the person's own browser knows:
the window's width, the zoom, the text size and spacing they chose
(`ui.text`, `ui.spacing`), the fonts, and whether it is a phone. So the page
measures itself where it is drawn (`design/base/29-measure.js`) and says
how it came out:

- each block's height, width and where its row starts;
- anything inside it that scrolls, with its content and box heights, or
  that is cut off where nothing scrolls, and how much is hidden;
- how far it runs past the right edge of the screen (WCAG 1.4.10);
- in a side pane that scrolls on its own, the pane's height;
- the screen: its width (phone below 600px, tablet below 1024px, desktop),
  pixel ratio, root font size, and how many columns the grid had.

It measures after the page loads and again on a real change of size (a
`ResizeObserver`, 4px or more), once the page has been still for a moment
and never more often than every five seconds, and sends only when
something changed. What scrolls by design says so with `data-scrolls`
(the conversation's history and its list of chats, a lookup's choices),
as `.sw-table-wrap` does for a wide table and fields and code do by being
what they are; it is counted apart and never flagged.

The server keeps the latest reading of each block on each kind of screen
(in memory and in the store's own notes, not synced to other copies, at
most 600, and dropped after seven days). A reading names the block as it
was drawn (its component, props, span, region, frame, size, and on a tab
which panes it has, but not its position): once any of those change, the
reading no longer stands and the block waits to be measured again.

Layout now then gives each row's heights ("Errands" collection 6
(820px)), says "Heights measured on your desktop (1,440px wide) and your
phone (390px wide)" or "Heights estimated: nobody has opened this tab since
it changed", judges blank space beside a shorter block by the measured
heights instead of by kind, and names what the screen could not show, each
with the `arrange_canvas` call that gives it room:

> "Errands" scrolls inside on your phone (1,240px of content in a 480px
> box): give it span 12: arrange_canvas {"blocks":[...]}

`GET /api/look` on Home, a tab or a block's own page gives the readings
under `measured` (and with scripts, the same measuring in the look
browser's window, `measured.look_browser`), and the accessibility runner
checks the seeded pages at desktop and phone widths with the same script.

When nothing is measured, Layout now keeps to the estimates. It does not
open a headless browser to measure on demand: that takes seconds on every
write, and its window is not the person's. A look with scripts measures
in passing, since its browser is already open.

### What is sent, and by whom

Only numbers and ids: block ids and versions, which page (tab or a
block's own), the tab's id, and sizes in whole pixels. Never a word a
block shows, a label, a value or an address. The script reads no text,
and a test holds it to that. `POST /canvas/measure` takes only those
fields, refuses anything else, caps every number (50,000px, 200 blocks)
and drops a block not on the page named or changed since it was drawn.

Only the people who may change a page are measured: the owner, editors
and hosts. A published page and someone who may only look get no
`data-measure` mark, so the script never runs for them, and the route
refuses them, an agent's key, and the internet. Nothing is sent when the
browser asks to save data (`navigator.connection.saveData`), or when a
program drives it (`navigator.webdriver`, a headless browser), since that
window is not a person's.

## Why a suggestion and not a solver

Work on AI layout pairs a model, which knows what the person meant, with
rules checked by code, which the model alone does not keep
([UI grammar](https://arxiv.org/abs/2310.15455),
[LayoutPrompter](https://arxiv.org/abs/2311.06495), and LLM plus MaxSAT
layout that guarantees UX rules). Sameway keeps the rules in code and
the decision with the model: the line says what breaks which rule and one
call that would fix it, and the model sends it, changes it, or leaves it.

## Not done

- Height is measured only once someone who may change a tab has opened
  it since it last changed; until then it is guessed from the component
  and its detail. Nothing measures a tab nobody opens.
- Readings are per kind of screen, not per person: the owner's phone and
  an editor's phone share one reading, the latest.
- A block's height also changes with its records (a list grows), which
  does not make a reading stale; it stands until the page is opened again
  or seven days pass.
- A block that scrolls sideways by design (a wide table in a narrow
  column) is counted but not flagged.
- "Most important" knows only what is late; it does not know what the
  person said matters most.
- Headings in the side panes are not part of the outline check.
