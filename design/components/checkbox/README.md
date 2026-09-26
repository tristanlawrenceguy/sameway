# checkbox

Use a checkbox for a single yes/no setting. The label says what "checked"
means in the affirmative ("Published", not "Not a draft"). For a choice
between several options use `select`.

## When not to use it

- To show a state nobody changes here: that is a `badge`.
- To tick a record done, pinned or shown where it is listed: that is `mark`,
  which saves itself in place.
- For a choice that acts at once, such as turning a part of the page on:
  that is a `button` with `pressed`, since a checkbox waits for the form.
- For two answers that are not plainly yes and no: two choices in a `select`
  say what each one means.

## Saying more

- `hint` is a line under the label, read with the box.
- `required` is only for a box that must be ticked to go on, such as agreeing
  to something. It says "(required)" in words, not with a star.
- `error` is said above the box, in words that say what to do, and the box is
  marked invalid and tied to it. The box keeps what the person chose.

The name is the label and never changes with the state: the box itself says
checked or not checked. The box is `--sw-size-box`, bigger than the browser's
own, and the label is part of the 44px target.
