# Motion

Motion exists to explain change: something arrived, something left,
something is different. It is never ambient.

## Tokens

| Token | Value | Use |
|---|---|---|
| `motion-fast` | 120 ms | Hover, press release, focus colour, the cross-fade under reduced motion |
| `motion-base` | 220 ms | Exits, page cross-fade |
| `motion-slow` | 420 ms | Enters, morphs |
| `motion-glow` | 1400 ms | The change glow |
| `motion-arrive` | 2600 ms | One block arriving: place, shape, content |
| `motion-between` | 3000 ms | From one arrival to the next |
| `motion-travel` | 1.5rem | How far a calendar's month slides (shared axis) |
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

## What a person does moves where it goes

Most motion follows the assistant's changes. A person's own actions used
to snap: a ticked task vanished from its group, a moved card reappeared
in another column, a filtered list was simply a new list. Now each is a
view transition that shows where the thing went
(`design/base/26-travel.js` and `26-travel.css`):

- **A tick** strikes the row through at once, its title greying over
  `motion-fast`. The row moves to Done when focus leaves the list, not
  under the person's hands (`13-mark.js`), and when it does it slides
  there, the groups and their counts settling around it. The same holds
  for a mark in a collection, and for whatever the assistant changes
  while the page follows a turn (`17-refresh.js`).
- **A board Move** glides the card to its new column across the reload,
  and focus comes back to its Move button as before.
- **Applying a list's filters or sort**, on a collection, the activity
  log or a calendar's kinds, moves the items that stay into their new
  order; those that go fade out where they were, new ones fade in.
- **A calendar's previous or next** slides the month in from the side
  asked for and the old one out the other way: one axis, `motion-travel`
  far, as Material's shared axis does.

How: the script names each list item (its block and its record) only for
the moment of a transition, on both pages of a navigation (`pageswap` on
the page left, `pagereveal` on the page arrived) or both sides of an
in-page change, and takes the names off after, so the page at rest is the
page the server sent and a refresh still compares blocks as sent.
Navigations to another page are left to the page cross-fade. A tick
stays a same-document change; the others are already form posts and
links, so they ride the cross-document transition the page already had,
with no script submitting forms (a form that works without scripts keeps
working the same way with them). `sameway.js` blocks the first render so
`pagereveal` is heard in time (Chrome's guidance; a deferred script alone
is not guaranteed to run before the first frame).

Timing: `motion-base` (220 ms) at the calm pace, `motion-fast` (120 ms)
at the quick pace, both inside NN/g's 0.1 to 1 s band for feeling direct
and Material's 200 to 300 ms for a change of this size. Transform and
opacity only, drawn by the browser from snapshots. A browser without view
transitions shows the finished page, with no error. Focus is never moved
by any of it. While a transition runs (a quarter of a second at most) the
page takes no clicks, which is the browser's rule and the reason these
stay short; nothing moves before the person has acted.

## Press: a control answers at once

Within a tenth of a second a response reads as caused by the press
(NN/g); later, as a press that did nothing. So every control answers in
the frame it is pressed, before any server has spoken
(`design/base/27-press.css`, `02-feedback.css`):

- **A button**, and a link shaped as one, rings and gives a little
  (scale 0.97) the moment it is pressed, and lets go over `motion-fast`.
- **A row or card that is one target** (a list's row, a type's listing,
  an agenda's day, a card's title) shades and is ringed inside its edge.
  Colour and a ring only: nothing changes place or size, so nothing
  around it moves.
- **A mark's tick draws itself**: the box fills and the check is drawn
  from its short stroke to its long one over `motion-base`. Checked is
  the shape of the tick as well as the fill; the box's edge is
  `border-strong`, 3:1.
- **A tick shows on the press.** The row is struck through at once, not
  when the server answers (`13-mark.js`); a refusal puts the box and the
  row back and says why as an alert. What is said is still the server's
  outcome, once, through the region there from the start: the box and
  the row show what was pressed, the words what happened.

Under reduced motion nothing gives or draws: the press is the colour and
the ring, the tick simply there. Under forced colours shadows and fills
go, so a pressed row is outlined in `Highlight`, and the mark's box, fill
and tick take the system's colours (`CanvasText`, `Highlight`,
`HighlightText`); no meaning rests on the animation.

## Arrival

A person is eased into new information, even when the change itself was
quick. A block added in the last turn arrives in three stages: first where
it will be (a dashed outline in the actor's colour over an empty slot, so the
place registers before anything else), then what it is (the frame is
uncovered), then what it says (the content fades in). Only then does it
glow. When several blocks changed in one turn they arrive one after another,
in the order they were made, with a pause between, so there is time to take
each one in: `motion-arrive` (2600 ms) for one block, `motion-between`
(3000 ms) from one start to the next.

An edited block is eased in too, in two stages rather than three, because
it is already there: an outline in the actor's colour marks the place that
is about to change, then the content crosses over (the old words dim, the
new ones settle), and then it glows. Edits take their turn in the same
sequence as additions.

The server sets `data-arrival` to each changed block's place in that order
(the conversation block never arrives; it is the person's own tool). The
stages are CSS only: no script is involved, and a browser that cannot
animate shows the finished page. Screen reader users are not made to wait:
the status region announces the turn's changes at once, and the content is
in the accessibility tree from the first moment.

## Pace, and showing everything at once

Someone who has already caught up should not have to wait. Any key or
click on the page ends the sequence and shows everything at once, and while
it runs a Show all button says so (`base/10-arrival.js`, an enhancement that
adds its own control and removes it when there is nothing left to show).
That keeps a sequence longer than five seconds within WCAG 2.2.2, which asks
for a way to stop content that updates on its own.

The pace itself belongs to the workspace, not a settings page: a person
tells the assistant "slower", "faster" or "no motion", and `set_pace` writes
`ui.pace` in `workspace.yaml`, which lands on the root as `data-pace`. calm
is the default; quick keeps the stages in a third of the time; still shows
everything at once, the way reduced motion does.

Under forced colours (Windows high contrast), box shadows and custom
colours are dropped by the browser, so the change marker becomes a
system-coloured outline and the arrival cover draws in system colours.

## The glow

The glow is the system's main provenance signal, and the reason the resting
interface carries no "added by" labels at all.

After a turn, the server marks blocks with `data-changed="added"` or
`"updated"`. Each blooms once in the colour of whoever changed it, holds
long enough to be found, and fades to nothing over `motion-glow` (1400 ms).
Human indigo, assistant teal.

A marker reports the exchange the page is showing and nothing older, so a
change never keeps announcing itself. Under reduced motion there is no
bloom; a static ring in the same colour stays for that page view instead, so
the change is still findable.

## Working state

While a request is in flight, `status/enhance.js` switches the status
component to `working`, whose dot pulses. The text changes at the same
time, so the state is announced and readable without the animation. During the
assistant's turn the words then follow what it is doing (below, "A turn
you can watch").

## Reduced motion

Under `prefers-reduced-motion: reduce`: every animation and transition
collapses to 0.01 ms, smooth scrolling is off, and the change marker
becomes a static 3px ring in the actor's colour. View transitions stay,
reduced rather than removed: nothing travels, grows or slides, and what
changed cross-fades where it now is over `motion-fast`. WCAG 2.3.3 counts
opacity as no motion, and Val Head's and Eric Bailey's advice is to
replace the movement that triggers vestibular symptoms (large travel,
zoom, parallax) with a cross-fade, not to take away the sign that
something changed. The still pace does the same, and so does a narrow
screen while the page follows a turn. Meaning is preserved; only movement
is removed.

## Rules

- Never animate colour alone to convey state; pair it with text.
- No looping animation except the working pulse, which stops when the
  request does.
- No animation longer than `motion-slow`, except the glow and the arrival
  stages, whose length is the point: they give a person time to take a
  change in, which is part of accessibility, not decoration.
- Do not add JavaScript to animate. If CSS cannot express it, it is not
  worth animating. A script may say what is the same thing before and
  after (a view transition name); the motion itself stays CSS.
- A person's own action moves in under a quarter of a second
  (`motion-base` or less), transform and opacity only, and never moves
  focus, or anything under the pointer before they act.
- Under reduced motion and the still pace a change may cross-fade; it may
  not travel, grow or slide. `tools/a11y-runner/motion.mjs` records every
  view transition animation of a tick, a Move, a filter and a month, and
  fails if one moves anything under reduced motion.

## The turn as it happens

A turn takes as long as the model and its tools take, and a person should
never be left looking at a page that does nothing. With scripts the
composer posts the same form to `/chat/stream` and the page shows the
turn as it goes: the person's message as recorded (and the box cleared
for the next one), a dot that says the model is thinking until anything
arrives, the reply's words as the model says them, each tool the moment
the model names it ("Adding a block", while the arguments are still
being written) filled in as it runs ("Adding a calendar") and ticked as
it lands, and each block the assistant made or changed arriving on the
canvas the moment it exists, with the same arrival stages as after a
reload. The log follows the turn only while the person is reading its
end: someone who has scrolled up or is selecting text is left where they
are. A message sent while the assistant is working waits and goes when
the turn is done. The turn itself runs to its end even if the page that
asked for it goes away, so a closed tab never leaves a change half made.
The rest of the page follows the turn as well: a record a collection or
a calendar shows, a block in a pane, a list that appears in the sidebar.
After each change and at the end, the page fetches itself as it now is
and moves what changed into place inside a view transition, so a block
that moved slides, a new one arrives and one that has gone leaves, with
the same motion as between navigations; the chat and every unchanged
block stay as they are, and the person keeps their scroll, focus and
caret (`design/base/17-refresh.js`). A closed tab never leaves a change half made;
a Stop control beside the status ends it on purpose, and the reply then
says it was stopped, with what was done kept. There is no limit on how
many tools a turn may use: it ends early only when it is plainly getting
nowhere (the same call again, or tools failing three rounds running),
and then in words, not an error. When several land at once they come
one at a time with a breath between (`motion-between` scaled to the
pace: none under `still` or reduced motion), so there is time to take
each in; a single change is not delayed. Without scripts the form posts
to `/chat` and the page comes back whole, as before. See
`design/base/14-live.js` and `internal/server/stream.go`.

## A turn you can watch

A turn is the longest wait in Sameway, and the page should show it is
alive without making noise (`design/base/28-turn.js`, `28-turn.css`):

- **The status line says what the assistant is doing**, in words its
  calls give: "Adding a chart of water…", "Looking up your tasks…",
  "Arranging the page…", "Changing the list of tasks…", "Writing the
  reply…" (`internal/chat/doing.go`). Not a spinner: a spinner says only
  that something is happening; the words say what. The status is a polite
  live region, so every change is read out; it changes at most once every
  2.5 seconds, the newest words winning when steps come close together,
  and the same words are never said twice in one turn. What is drawn is
  what is heard: the words are the region's own. A step still going after
  fifteen seconds says so in its own words ("Still adding a chart of
  water…"). The page following the turn does not put the server's
  "working" back over the step being said.
- **The reply grows calmly.** Its words go on the page a few times a
  second (every 80 ms), not letter by letter and with no caret, and the
  log follows the end at once rather than gliding after every word. The
  words are not in a live region: a screen reader hears the finished
  reply once, at the end, with the status (`swSay`).
- **A block about to be added holds its place**: as soon as the model
  starts the call, a dashed outline in the assistant's colour stands where
  the block will land, as wide as it will be. It is the first stage of the
  block's arrival, held; when the block lands it takes that place and its
  arrival plays on from the outline. A block about to be changed is
  outlined, solid (dashed if it is about to go). The held place says
  "Adding a chart of water…" to a screen reader that reaches it.

Nothing travels. The outline fades in over `motion-slow`. Under reduced
motion it is simply there, and the block that lands cross-fades in over
`motion-fast` where the outline was instead of its staged arrival; under
the still pace both are simply there. Under forced colours the outlines
take `Highlight`.

## Sources

- Material 3 motion: durations step by 50 ms, short 50 to 200 ms, medium
  250 to 400 ms; standard easing; shared axis X slides 30dp and fades
  over 300 ms ([Material Components, Motion](https://github.com/material-components/material-components-android/blob/master/docs/theming/Motion.md),
  [shared axis](https://blog.stylingandroid.com/material-motion-shared-axis/)).
- Apple HIG: motion with a purpose, brief feedback, never the only way
  something is conveyed, subtler under Reduce Motion
  ([Motion](https://developer.apple.com/design/human-interface-guidelines/motion)).
- Val Head: large travel, motion out of step with scrolling, and zoom are
  the triggers; reduce rather than remove, cross-fade instead
  ([Designing safer web animation](https://alistapart.com/article/designing-safer-web-animation-for-motion-sensitivity/),
  [reduced motion in the wild](https://valhead.com/2020/05/09/reduced-motion-in-the-wild/)).
  Eric Bailey: reduce, don't remove, since animation helps comprehension
  ([Revisiting prefers-reduced-motion](https://css-tricks.com/revisiting-prefers-reduced-motion/)).
  [web.dev on prefers-reduced-motion](https://web.dev/articles/prefers-reduced-motion).
  Vestibular disorders: avoid parallax, large travel, zoom, spinning
  ([A11y Project](https://www.a11yproject.com/posts/understanding-vestibular-disorders/)).
- WCAG 2.2: [2.2.2 Pause, Stop, Hide](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html)
  (more than 5 s needs a stop: the arrival's Show all),
  [2.3.1 Three Flashes](https://www.w3.org/WAI/WCAG22/Understanding/three-flashes-or-below-threshold.html)
  (nothing here flashes), [2.3.3 Animation from Interactions](https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html)
  (motion from an interaction can be turned off; opacity is not motion).
- View transitions: same-document in Chrome 111, Safari 18, Firefox 144;
  cross-document in Chrome 126 and Safari 18.2; `view-transition-class` in
  Chrome 125, Safari 18.2, Firefox 144; a page takes no clicks while one
  runs; `pagereveal` needs a script that runs before the first render
  ([Chrome, same-document](https://developer.chrome.com/docs/web-platform/view-transitions/same-document),
  [cross-document](https://developer.chrome.com/docs/web-platform/view-transitions/cross-document),
  [MDN](https://developer.mozilla.org/en-US/docs/Web/API/View_Transition_API),
  [Bramus on interactivity](https://www.bram.us/2025/01/29/view-transitions-page-interactivity/)).
- FLIP, which view transitions do for us: measure first and last, invert
  with a transform, play; transform and opacity only
  ([Paul Lewis](https://aerotwist.com/blog/flip-your-animations/)).
- Response times: 0.1 s feels instant, 1 s keeps the flow; small feedback
  about 100 ms, a change on screen 200 to 300 ms
  ([NN/g, response times](https://www.nngroup.com/articles/response-times-3-important-limits/),
  [NN/g, animation duration](https://www.nngroup.com/articles/animation-duration/)).

- Status said in words, politely, and not too often: a status message is
  heard without moving focus ([WCAG 4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html));
  live regions announce every change, queue behind one another and are
  best kept few and terse ([Scott O'Hara, Are we live?](https://www.scottohara.me/blog/2022/02/05/are-we-live.html),
  [MDN, ARIA live regions](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Guides/Live_regions)).

## Not done, and why

- **Text that types itself.** Drawing the reply letter by letter, or a
  blinking caret at its end, is motion that explains nothing: the words
  are already there. They land in short bursts instead.
- **Saying every step.** A turn can call ten tools in a few seconds; a
  screen reader told each would still be reading the first ones after the
  turn was over. The newest step is said, and nothing twice.
- **A spinner in the status.** The dot pulses four beats and stops (WCAG
  2.2.2); the words carry the state.
- **A held place in a pane.** The place is held only on the main canvas,
  where the block lands at once; a block for a pane arrives with the page
  that follows the turn.

- **Moving a ticked row at once.** The next row would slide up under the
  pointer or the next Space; it moves when focus leaves the list.
- **Filters submitted by script, for a same-document transition.** The
  forms are GETs that work without scripts and already ride the
  cross-document transition; submitting by script would add nothing a
  person sees.
- **Table rows.** A row cannot be lifted out of its table to move; a list
  shown as a table changes in place under the page cross-fade.
- **Blocks resizing faster.** A block that grows because of a person's
  action still morphs over `motion-slow`, as it does for the assistant's
  changes; the items inside it travel in `motion-base`.
- **Calm meaning less travel.** Calm is the default pace, so making it a
  cross-fade would take the motion from nearly everyone; the still pace
  and reduced motion are the ways to ask for less, and both cross-fade.
- **Firefox across pages.** Firefox has same-document transitions (a
  tick slides) but not cross-document ones yet; there a Move, a filter
  or a month shows the finished page.
- **Optimistic changes beyond a tick.** A board Move, an edit and a
  delete come back as a page, with their outcome and focus where they
  were; showing them before the server agrees would mean undoing a
  change on screen that the person may already have read as done. A
  tick is the one place the press itself is the whole change.
- **A tick that springs or bounces.** The check draws once, in the time
  of a fade; anything more is decoration.
- **Letting clicks through during a transition**
  (`::view-transition { pointer-events: none }`). A click would land on a
  page the person cannot see yet; a quarter of a second without clicks is
  the safer cost.
