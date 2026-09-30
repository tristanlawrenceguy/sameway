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
| Tall things get a row or a pane of their own; glance and brief sizes go in panes | A short card beside a tall calendar leaves a hole under the card | this design system's grid (`design/base/06-layout.css`) |

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

## Why a suggestion and not a solver

Work on AI layout pairs a model, which knows what the person meant, with
rules checked by code, which the model alone does not keep
([UI grammar](https://arxiv.org/abs/2310.15455),
[LayoutPrompter](https://arxiv.org/abs/2311.06495), and LLM plus MaxSAT
layout that guarantees UX rules). Sameway keeps the rules in code and
the decision with the model: the line says what breaks which rule and one
call that would fix it, and the model sends it, changes it, or leaves it.

## Not done

- Height is guessed from the component and its detail, not measured.
- "Most important" knows only what is late; it does not know what the
  person said matters most.
- Headings in the side panes are not part of the outline check.
