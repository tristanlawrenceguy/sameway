# Elevation

Depth says which surfaces are which: the page, the things on it, the
thing that can be pressed, the thing being made. It is never decoration,
and it never carries meaning alone: every surface that matters has an
edge (a border) as well as a shadow, because forced colours take shadows
away and some people cannot see a soft shadow at all.

## Levels

| Level | Light | Dark | Use |
|---|---|---|---|
| 0, the page | `bg` | `bg` | The page itself |
| 1, at rest | `bg-raised`, `border`, `shadow-1` | `bg-raised`, `border`, `shadow-1` (a faint light rim) | A block, a card, a message |
| 2, lifted | `bg-lift`, `border-lift`, `shadow-2` | `bg-lift` (lighter), `border-lift` (lighter), `shadow-2` | A card that can be pressed, under the pointer or keyboard focus; a block being worked with |
| 3, over | `shadow-3` | `shadow-3` with a lighter rim | What floats over the page: a menu, a popover |

Shadows are layered, the way light falls in a room: a short, darker key
shadow close under the surface and a long, faint ambient one around it
(`shadow-1` to `shadow-3`). Corners are rounder than before (`radius-sm`
6px, `radius-md` 10px, `radius-lg` 16px): soft enough to read as objects
on a surface, not so round that a dense list of cards reads as pills.

## Dark is lighter, not darker

On a dark page a shadow has almost nothing to be darker than, so it hardly
shows. Material's dark theme answers with surfaces that get lighter the
higher they are, as if nearer a light; Material 3 does the same with named
surface tones (surface container lowest to highest) instead of shadows.
Sameway does both, in tokens: in dark, `bg-raised` is a step lighter than
the page and `bg-lift` a step lighter again, `border-lift` is a lighter
edge, and every shadow carries a faint light rim (`0 0 0 1px` white at a
few percent) so the outline of a surface shows where its shadow cannot.
Text keeps 7:1 on every surface (`internal/tokens/contrast_test.go` has
the lifted surface too).

## A card that can be pressed lifts

A card that opens wherever it is pressed (`.sw-card[data-press]`), and any
card whose link has keyboard focus, lifts under the pointer or focus: its
surface and edge lighten, its shadow deepens to `shadow-2`, and it rises
one pixel, by a transform, so nothing around it moves. Under reduced
motion it does not rise; the surface, edge and shadow still change. A
block lifts its shadow and edge but not itself: a block is not one target.

## Something on its way

Where something takes noticeable time, its shape stands in its place
rather than empty space: a block the assistant is about to add is a few
grey bars in the outline of where it will land (`.sw-skeleton`, see
motion.md, A turn you can watch). The bars may shimmer, very slowly, a
light passing across them twice (4.8 s, under WCAG 2.2.2's five
seconds), then they are still. Under reduced motion and the still pace
they never shimmer. The bars are hidden from screen readers; the place
says what is coming in words ("Adding a chart of water…") and is
`aria-busy` until it lands, as is a block being changed.

## Forced colours

Windows high contrast and other forced colours drop every shadow and
custom background. So every level carries a border, which forced colours
draw in the system's colour (`CanvasText`); a lifted card's edge becomes
`Highlight`; skeleton bars are drawn in `GrayText` so the shape still
shows. Nothing about depth is lost that a border cannot say.

## Sources

- Material Design 2, dark theme: lighter surfaces for higher elevation,
  dark grey not black, shadows weak on dark
  ([Dark theme](https://m2.material.io/design/color/dark-theme.html),
  [Material Components for Android, Dark theme](https://github.com/material-components/material-components-android/blob/master/docs/theming/Dark.md)).
- Material 3 replaced elevation overlays with tonal surface roles
  (surface container lowest to highest)
  ([Flutter, new ColorScheme roles](https://docs.flutter.dev/release/breaking-changes/new-color-scheme-roles),
  [Material 3 in Compose](https://developer.android.com/develop/ui/compose/designsystems/material3)).
- Skeleton screens: a page-shaped placeholder makes a wait feel shorter
  than a spinner, for waits of a few seconds, and should look like what
  comes ([NN/g, Skeleton screens 101](https://www.nngroup.com/articles/skeleton-screens/)).
- [WCAG 2.2.2 Pause, Stop, Hide](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html):
  movement that starts by itself and lasts more than five seconds needs a
  way to stop it; the shimmer stops itself first.
- Forced colours set `box-shadow` to none and replace colours with the
  system's ([MDN, forced-colors](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/forced-colors)).

## Not done, and why

- **A block that rises.** A block holds controls and is not one target;
  rising would say it can be pressed.
- **A shimmer that runs until the content comes.** A turn can take a
  minute; a light passing for that long is motion that explains nothing
  after the first pass, and WCAG 2.2.2 asks for it to stop.
- **Skeletons for pages and charts the server draws.** A chart is summed
  on the server before the page is sent, so there is no moment in the
  browser when it is missing; a page that comes back whole needs no
  stand-in.
- **Tinting lifted surfaces with the accent** (Material 3's tonal
  colour). One blue is the brand's only colour (style.md); a lifted card
  in blue would read as selected.
- **Glass and blur behind surfaces.** Text over a blurred page cannot be
  held to 7:1, and blur costs a phone dearly.
