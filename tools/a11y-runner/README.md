# Browser tests (dev only)

Node is used only here, never at runtime. Three scripts, all headless Chromium:

| Script | What it proves | Needs |
|---|---|---|
| `npm run test:axe` | Every component example passes axe-core at WCAG 2.2 AA; AAA findings are printed as warnings. | nothing |
| `npm run test:keyboard` | Every focusable element in every example is reachable by Tab in DOM order with a visible focus ring, and each component works the way its manifest keyboard map says. | nothing |
| `npm run test:pages` | A running server can be driven by a keyboard-only person and by an agent using roles, accessible names, and `/api/describe`, and every page is axe clean. | `sameway serve` with `llm.provider: none`, URL in `SAMEWAY_URL` |

```bash
cd tools/a11y-runner
npm install            # also downloads Chromium
npm test               # axe + keyboard
SAMEWAY_URL=http://127.0.0.1:8080 npm run test:pages
```

`shell.mjs` is the page wrapper the first two scripts render examples into.
It carries the real tokens and base CSS plus an h1 and h2 so components that
default to heading level 3 sit in a valid outline.

To waive an AAA rule for one component, add `examples/a11y-waivers.json`
mapping the axe rule id to a reason. AA rules cannot be waived.
