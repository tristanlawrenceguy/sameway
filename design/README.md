# Sameway design system

Accessible HTML patterns that humans and AI agents navigate the same way.
This folder is published on its own as `@sameway/design` and is also embedded
in the `sameway` binary.

## What is in here

- `tokens/tokens.json` is the single source of design tokens. `tokens.css` is
  generated from it with `go run ./tools/tokens` and must be checked in.
- `base/base.css` is the reset, typography, focus ring, motion, and page skeleton.
- `components/<name>/` is one component per folder:
  - `manifest.json` describes props (JSON Schema), the accessibility contract,
    the keyboard map, and how a machine identifies and operates it.
  - `template.html` is a Go `html/template` that receives the validated props.
  - `style.css` is the component's CSS. Only tokens, no raw colors.
  - `examples/*.html` are the rendered examples. They are the standalone spec
    and the golden output the template must match exactly.
  - `README.md` explains when to use it.

## Using it without the binary

Load `tokens/tokens.css`, `base/base.css`, and the `style.css` of the
components you need, then copy the markup from `examples/`. Every example is
self-contained and WCAG 2.2 AA clean, AAA where the manifest says so.

## Rules for a new component

Run `sameway component new <name>` to scaffold. A component is not done until:

1. The manifest has a full props schema and an accessibility contract.
2. Every example renders from the template with zero diff (`make golden`).
3. `make a11y` reports no AA violations for its examples.
4. It uses tokens only, honours `prefers-reduced-motion`, and keeps every
   interactive target at least 44 by 44 CSS pixels.
