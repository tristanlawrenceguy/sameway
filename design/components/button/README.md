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
