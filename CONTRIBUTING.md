# Contributing

Thank you. Sameway is built so that people and AI agents can both contribute
safely, which means the rules are mechanical and the tooling enforces them.

## Before you open a pull request

```bash
make check     # gofmt, go vet, repo lint, tests
```

If you changed a component template or manifest, also run `make golden` and
commit the regenerated example files. If you have Node available, run
`make a11y`; CI runs it regardless.

## What we accept

- Components that meet the accessibility contract in `design/README.md`.
- Field types, generators, and surfaces that derive from the schema and
  manifests rather than adding per-type code.
- Documentation fixes, example workspaces, and tests.

## What we push back on

- Client-side rendering, frontend frameworks, or JavaScript that a component
  needs in order to work at all.
- Files over 300 lines, or files that do more than one thing.
- Colour used as the only signal, targets under 44 pixels, missing labels.
- Dependencies that duplicate the standard library.

## AI-written contributions

Welcome, and expected. Point your agent at `AGENTS.md`. A human reviews every
pull request; keep each one to one change so review stays possible.

## Reporting accessibility problems

Open an issue with the component name, the assistive technology or input
method you used, and what you expected. Accessibility bugs are treated as
correctness bugs.
