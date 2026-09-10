# The quiet layer

Accessible does not mean permanently labelled. An interface that shouts
"Added by assistant at 20:24" on every block, with an Edit and a Remove
button beside it, is noisy for everyone and no more usable for anyone.

So Sameway separates **being available** from **being visible**.

## The rule

Chrome is present for everyone at all times, and visible on demand.

Present means: in the DOM, in the tab order, in the accessibility tree, with
a real accessible name, hittable at its own coordinates. Screen reader users
and agents have it continuously. It is faded to `opacity: 0` until the group
it belongs to is hovered, or something inside that group takes focus.

What is never used for this: `display: none`, `visibility: hidden`, the
`hidden` attribute, `aria-hidden`, `inert`, or `tabindex="-1"`. Every one of
those removes the control from screen reader users too, which is the
opposite of the goal. A test asserts that none of them appear in the quiet
layer.

```html
<li class="sw-block sw-reveal">
  … the component …
  <div class="sw-bar sw-quiet">
    <span class="sw-badge">Added by assistant at 20:24</span>
    <a class="sw-link">Edit<span class="sw-visually-hidden"> card</span></a>
    <button class="sw-button">Remove<span class="sw-visually-hidden"> card</span></button>
  </div>
</li>
```

`.sw-reveal` marks the group. `.sw-quiet` marks what fades. Layout never
shifts, because the bar is positioned and the space is always reserved.

## Always on where fading would strand someone

- coarse pointers and touch screens, which have no hover
- `prefers-contrast: more`
- forced-colours modes, where opacity is unreliable
- workspaces with `ui.controls: visible` in `workspace.yaml`

## Compact labels, complete names

A control in a quiet bar shows one word. The rest of its name is real text
in a visually hidden span, so the accessible name stays unique and
meaningful: "Edit" reads as "Edit card", "Remove" as "Remove table". That is
the `context` prop on button and link. It matters because a screen reader
user listing the buttons on a page, and an agent targeting one by role and
name, both need to tell them apart.

## What stays visible

Permanent, tiny, and cheap: the actor rail down the left edge of each block,
in the colour of whoever last touched it. Enough to scan a page and see who
built what, without a word of chrome. The readable version of the same fact
is one hover or one Tab away, and it is always in the accessibility tree.

This is the one place where colour alone carries information visually by
default. It is supplementary, never the only route: the same provenance is
in the badge text, in the change receipt under the assistant's reply, and in
the activity log. Someone who wants it on screen permanently sets
`ui.controls: visible`.

## Progressive disclosure

Detail that most people do not need most of the time goes in a
`disclosure`: closed by default, with a summary that says what is inside and
how many items, so nobody has to open it to find out.

The content of a closed `<details>` is not in the accessibility tree. That
is what makes it quiet, and it is the one case where hiding really does hide
it from everyone equally. So anything behind a disclosure is also somewhere
else in full: the activity log is at `/activity` and `/api/activity`.

## For agents

Nothing special is required. Query and click as usual.

```
[data-block-id] .sw-bar button      per-item controls, always present
role=button, name="Remove card"     the way to target one
details[data-component=disclosure]  closed detail; click the summary to open
```

Browser automation reports `opacity: 0` elements as visible, because
visibility checks look at layout and `visibility`, not opacity. A faded
control is clickable without hovering first.
