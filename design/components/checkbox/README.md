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

## Why it works this way

- **Ticks where it is.** With scripts on, a mark saves in the background
  and the person stays on the box they ticked, instead of the page
  reloading to its top; without scripts the form is sent as before
  ([WCAG 3.2.2](https://www.w3.org/WAI/WCAG22/Understanding/on-input.html)).
- **One name whatever its state.** The box says checked; a name that also
  changes ("Mark done", then "done") says the state twice or wrongly
  ([APG checkbox](https://www.w3.org/WAI/ARIA/apg/patterns/checkbox/)).
- **A word in sight.** On a record's page the box has its word beside it,
  Done or Pinned, not a box with no visible label
  ([WCAG 3.3.2](https://www.w3.org/WAI/WCAG22/Understanding/labels-or-instructions.html)).
- **A bigger box**, with the label part of a 44px target
  ([GOV.UK checkboxes](https://design-system.service.gov.uk/components/checkboxes/)).
- **Errors in words above the box**, tied to it, and the person's choice
  kept ([WCAG 3.3.1](https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html)).

Not done, and why: a star for required (not everyone knows it, and a
screen reader may skip it); a switch for yes and no (it looks like it acts
at once, which a checkbox in a form does not).
