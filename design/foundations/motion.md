# Motion

Motion exists to explain change: something arrived, something left,
something is different. It is never ambient.

## Tokens

| Token | Value | Use |
|---|---|---|
| `motion-fast` | 120 ms | Hover, press, focus colour |
| `motion-base` | 220 ms | Exits, page cross-fade |
| `motion-slow` | 420 ms | Enters, morphs, the change flash |
| `motion-ease` | `cubic-bezier(0.2, 0.8, 0.2, 1)` | Everything that moves |
| `motion-ease-out` | `cubic-bezier(0, 0, 0.2, 1)` | Flashes that only fade |

## Cross-document view transitions

Sameway pages are full-page navigations. `base.css` opts the document into
cross-document view transitions (`@view-transition { navigation: auto }`),
so when a form submits and the page reloads, the browser morphs elements
that exist on both pages and animates the ones that appeared or vanished.
Each message and canvas block carries a unique `view-transition-name` and
the `sw-vt-item` class, which maps to:

- new items enter with `sw-enter` (fade, 8px rise, slight scale)
- removed items leave with `sw-exit`
- moved or resized items morph over `motion-slow`

Browsers without the feature simply reload, and the change markers below
still play, so nothing is lost.

## Change markers

After a turn, the server marks blocks with `data-changed="added"` or
`"updated"`. They flash once on load in the actor's colour (`sw-flash`),
then rest. This is how a person sees at a glance what the assistant just
did, and how an agent can query it.

## Working state

While a request is in flight, `status/enhance.js` switches the status
component to `working`, whose dot pulses. The text changes at the same
time, so the state is announced and readable without the animation.

## Reduced motion

Under `prefers-reduced-motion: reduce`: view transitions are disabled,
every animation and transition collapses to 0.01 ms, smooth scrolling is
off, and the change marker becomes a static 3px ring in the actor's colour.
Meaning is preserved; only movement is removed.

## Rules

- Never animate colour alone to convey state; pair it with text.
- No looping animation except the working pulse, which stops when the
  request does.
- No animation longer than `motion-slow`.
- Do not add JavaScript to animate. If CSS cannot express it, it is not
  worth animating.
