# button

Use a button for an action on the current page: submit, save, delete, send.
Use `link` when the result is a different page. The label should say what
happens ("Save note"), not just "OK". Prefer validating on submit over
disabling the button, because a disabled button leaves the tab order and gives
no feedback about why. See `manifest.json` for props and the accessibility
contract.

## Actions

A button given `action` (the id of an action record) is rendered inside
its own form posting to `/act/<id>`, with `from` as the page to return to.
Pressing it runs the action: a webhook to a URL the person set up, an
arrangement, or a message to the assistant. It is a real form, so it works
without a script, and it must not be placed inside another form.
