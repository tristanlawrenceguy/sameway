# button

Use a button for an action on the current page: submit, save, delete, send.
Use `link` when the result is a different page. See `manifest.json` for props
and the accessibility contract.

## Labels

Say what happens, as a verb and what it acts on, in sentence case: "Save
note", "Delete workspace", never "OK" or "Yes". When the visible word stands
alone ("Copy", "Remove") give `context`, read after it and not shown, so it
is heard as "Remove card".

## Kinds

One primary action per place, the thing most people came to do. Secondary
for the others. Danger only for what cannot be taken back, and behind a
confirmation such as typing the name. Quiet for per-item controls: in a
control bar the bar is its frame; on its own it keeps a strong edge, so it
still reads as a button.

## Sent once

A form is sent once. When a button submits a form, it is marked busy
(`aria-disabled`) and further presses are ignored until the next page
arrives: a double press from a slow connection, a habit or a tremor does not
create or delete twice. It keeps its focus and its words. A form a script
sends itself, and a search, are left alone. The server should still be safe
against a repeat.

## Not disabled

Prefer checking on submit over disabling a button: a disabled button gives
no reason, is hard to read, and drops out of reach. Use `disabled` only
where research shows it helps.

## On and off

A button that turns something on and off takes `pressed`, which renders
`aria-pressed` and a pressed-in look. Keep its label the same either way
("Show done tasks"); a label that changes ("Turn on", "Turn off") takes no
`pressed`.

## Actions

A button given `action` (the id of an action record) is rendered inside
its own form posting to `/act/<id>`, with `from` as the page to return to.
Pressing it runs the action: a webhook to a URL the person set up, an
arrangement, or a message to the assistant. It is a real form, so it works
without a script, and it must not be placed inside another form.

## Why it works this way

- **Sent once.** The pressed button is busy until the next page arrives,
  not for a set second, since a slow request would then go twice. It is let
  go after 15 seconds for a reply that never leaves the page, and on Back
  ([GOV.UK button](https://design-system.service.gov.uk/components/button/)).
- **Busy, not disabled.** `aria-disabled` keeps the button in reach with
  its name; a disabled one gives no reason and drops out of reach.
- **A quiet button keeps an edge** on its own, so Cancel beside Save still
  reads as something to press
  ([NN/g on clickable elements](https://www.nngroup.com/articles/clickable-elements/)).
- **On and off not by colour alone.** `pressed` gives `aria-pressed` and a
  pressed-in look, with the label kept the same
  ([Inclusive Components toggle buttons](https://inclusive-components.design/toggle-button/)).
- **44px targets**, in bars too
  ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).

Not done, and why: a guard of one second (a slow request is sent again
after it); disabling the button while it sends (it loses focus and its
value goes missing from the form); guarding searches (they are harmless to
repeat).
