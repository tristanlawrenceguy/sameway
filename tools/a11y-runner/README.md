# Browser tests (dev only)

Node is used only here, never at runtime. All headless Chromium:

| Script | What it proves | Needs |
|---|---|---|
| `npm run test:axe` (`run.mjs`) | Every component example passes axe-core at WCAG 2.2 AA and AAA in light and dark mode; its tree holds the role its manifest declares; it reflows at 320px, keeps all its text with WCAG text spacing and at 200% text, and stops moving under reduced motion; field edges are 3:1, targets reach 44x44, state is never colour alone, controls show a name that is in their accessible name, marks survive forced colours, and errors are tied to their fields. | nothing |
| `npm run test:keyboard` (`keyboard.mjs`) | Every focusable element in every example is reachable by Tab in DOM order with a focus ring in light, dark and forced colours, 2px and 3:1 against what it is drawn over; Shift+Tab walks back and nothing traps focus; every control works by keyboard the way its kind promises; the manifest keyboard map says what Tab reaches. | nothing |
| `npm run test:pages` (`pages.mjs`, then `site.mjs`) | A running server can be driven by a keyboard-only person and by an agent using roles, accessible names, and `/api/describe`. Then every page, and a canvas holding every component example in each region and size, gets the same checks, a Tab walk at desktop, at 320px and on a phone held sideways for focus hidden under sticky parts, live regions that can announce, one place per link name, and `/api/look`; and the site: every page titled after itself and no two alike, the main navigation the same everywhere, and every form with a required field saying what is wrong when sent empty. | `sameway serve` with `llm.provider: none`, URL in `SAMEWAY_URL` |

```bash
cd tools/a11y-runner
npm install            # also downloads Chromium
npm test               # axe + keyboard
SAMEWAY_URL=http://127.0.0.1:8080 npm run test:pages
```

`shell.mjs` is the page the component suites render an example into: the
real tokens, base and component CSS, the page's column and gutter, and an h1
and h2 so components that default to heading level 3 sit in a valid
outline. `checks.mjs` holds the checks for modes, reflow, spacing, motion
and hidden focus; `visual.mjs` the ones axe does not make (focus
appearance, field edges, target size, colour-only state, visible names,
forced colours, text zoom, error wiring).

Forced colours is checked for focus rings and for marks drawn only in
background colour, not for contrast: there the person's own theme sets
every colour. On live pages the quiet layer is shown before axe runs,
because axe passes over anything at opacity 0 and those controls are what
hover and focus show. Checks run with reduced motion so transitions and
arrivals are finished before anything is measured.

## Waivers

To waive an AAA rule for one component, add `examples/a11y-waivers.json`
mapping the axe rule id to a reason. `target-size-44` waives the 44px
target for a component that cannot meet it, with the reason; it is the
only non-axe key. AA rules cannot be waived.
