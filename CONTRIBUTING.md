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

## Your first contribution: a component

A component is one folder in `design/components/<name>/`, and it is the easiest place to start:

- `manifest.json`: its props, its accessibility contract, its keyboard map, when to use it and when not to
- `template.html`: how it renders
- `style.css`: its styles, using design tokens only
- `examples/`: the cases it must render, one file each
- `README.md`: a short note for people

Copy the closest existing component (`go run ./cmd/sameway component new <name>` scaffolds one), change it, then run:

```bash
make golden   # writes the example output files
make check    # formatting, lint, and the component contract tests
```

You are done when `make check` passes. The tests check what a reviewer would otherwise have to: every control has a name, ids are unique, labels point at real fields, CSS uses tokens, hostile props stay inert, and every example renders exactly as committed. `make a11y` runs axe and real keyboard tests if you have Node; CI runs it either way.

Issues labelled [good first issue](https://github.com/tristanlawrenceguy/sameway/labels/good%20first%20issue) list components we would like to have.

## How pull requests get reviewed

Much of Sameway is built by AI agents working through the same checks you run, which is why the history moves quickly. Outside pull requests are read by a person, not merged by a bot. Keep each one to one change and say what you tested.

## What we accept

- Components that meet the accessibility contract in `design/README.md`.
- Field types, generators, and surfaces that derive from the schema and
  manifests rather than adding per-type code.
- Documentation fixes, example workspaces, and tests.

## What we push back on

- Client-side rendering, frontend frameworks, or JavaScript that a component
  needs in order to work at all.
- Functions over 80 lines, files over 400 lines, files that do more than one
  thing, or files split by size (`*_more.go`, `*_extra.go`, `*_helpers.go`).
- Colour used as the only signal, targets under 44 pixels, missing labels.
- Dependencies that duplicate the standard library.

## AI-written contributions

Welcome, and expected. Point your agent at `AGENTS.md`. A human reviews every
pull request; keep each one to one change so review stays possible.

## Reporting accessibility problems

Open an issue with the component name, the assistive technology or input
method you used, and what you expected. Accessibility bugs are treated as
correctness bugs.
