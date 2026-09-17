# Colour

White, one ink at two strengths, one blue, plus the assistant's teal and
four state tones (see [style.md](style.md)). Every text pairing meets WCAG 2.2 AAA (7:1) on every surface it can
appear on, in both themes. `internal/tokens/contrast_test.go` lists the
pairings and fails the build when one slips.

## Surfaces

| Token | Light | Dark | Use |
|---|---|---|---|
| `bg` | `#ffffff` | `#0f1117` | Page |
| `bg-muted` | `#f2f2f4` | `#1e222c` | Quiet regions, hover on a row, table heads |
| `bg-raised` | `#ffffff` | `#171a22` | Cards, blocks, panels, on one soft shadow |
| `border` / `border-strong` | `#e3e3e8` / `#85868c` | `#343a48` / `#7f8798` | Hairlines / control edges (3:1) |

## Text

`fg` for body, `fg-muted` for secondary. Both reach 7:1 on all three surfaces.

## Actor tones: who did it

| Actor | Token | Light | Dark | Where it appears |
|---|---|---|---|---|
| A person | `human` | `#1a45a8` blue | `#b8bfff` | Their messages, blocks they edited, their activity |
| The assistant | `assistant` | `#075a44` teal | `#6ee7bf` | Its messages, blocks it made, its activity |
| The software | `system` | `#703e00` amber | `#f5c26b` | Error notices, log entries about failures |

Each has a `-soft` tint for backgrounds; the tone's text colour reaches 7:1
on its own tint. Set `data-actor` on any element and components inside it
read `--sw-actor` and `--sw-actor-soft`. The accent (links, primary buttons)
is the human blue: the interface belongs to the person.

## State tones

`success`, `warning` (same as system amber), `danger`, `info`, each with a
`-soft` tint. Used by alert, badge, status, and invalid form fields.

## Focus

`focus` is a warm orange ring, `focus-halo` is the page colour drawn inside
it, so the double ring reads on any surface including a primary button.

## Changing the palette

Edit `tokens/tokens.json`, run `go run ./tools/tokens`, run
`go test ./internal/tokens/`. The test output names the pairing that fails
and the ratio it reached.
