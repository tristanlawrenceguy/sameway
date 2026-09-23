# Browser tests (dev only)

Node is used only here, never at runtime. Three scripts, all headless Chromium:

| Script | What it proves | Needs |
|---|---|---|
| `npm run test:axe` | Every component example passes axe-core at WCAG 2.2 AA in light and dark mode, reflows at 320px wide without sideways scrolling, keeps all its text with WCAG text spacing forced on, and stops moving under reduced motion. AAA findings are printed as warnings. | nothing |
| `npm run test:keyboard` | Every focusable element in every example is reachable by Tab in DOM order with a visible focus ring in light, dark and forced colours; Shift+Tab walks back and nothing traps focus; every control works by keyboard the way its kind promises (links, buttons, summaries, checkboxes, selects, date and file fields, text fields and textareas); and the manifest keyboard map says what Tab reaches. | nothing |
| `npm run test:pages` | A running server can be driven by a keyboard-only person and by an agent using roles, accessible names, and `/api/describe`. Then every page, and a canvas holding every component example in each region and size, is checked with the same axe, reflow, spacing and motion checks, walked with Tab at desktop and 320px for focus hidden under sticky parts, and read through `/api/look` for problems. | `sameway serve` with `llm.provider: none`, URL in `SAMEWAY_URL` |

```bash
cd tools/a11y-runner
npm install            # also downloads Chromium
npm test               # axe + keyboard
SAMEWAY_URL=http://127.0.0.1:8080 npm run test:pages
```

`shell.mjs` is the page wrapper the first two scripts render examples into.
It carries the real tokens and base CSS, the page's column and gutter, and
an h1 and h2 so components that default to heading level 3 sit in a valid
outline. `checks.mjs` holds the in-page checks the suites share.

Forced colours is checked for focus rings only: there the person's own
theme sets every colour, so contrast is not the page's to meet. On live
pages the quiet layer is shown before axe runs, because axe passes over
anything at opacity 0 and those controls are what hover and focus show.

To waive an AAA rule for one component, add `examples/a11y-waivers.json`
mapping the axe rule id to a reason. AA rules cannot be waived.
