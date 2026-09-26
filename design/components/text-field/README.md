# text-field

Use a text field for one line of input. The label is always visible. Put help
in `hint`, not in placeholder text. On a validation failure re-render with the
submitted `value` and an `error` that says how to fix it. Shares its
stylesheet classes with `textarea` and `select`.

## Why it works this way

- **An error says Error**, hidden before the words, as the error summary's
  and alerts' do, so it is heard as one; and the field has a bar at its edge
  as well as the red border, which forced colours take away
  ([GOV.UK error message](https://design-system.service.gov.uk/components/error-message/)).
- **Numbers in a text field** with a number keyboard, not the browser's
  number field, whose wheel and arrows change the value by accident and
  which quietly drops what it cannot read; the server says what is wrong
  instead ([GOV.UK text input](https://design-system.service.gov.uk/components/text-input/)).
- **The count says too many** when the page comes back over the limit, and
  counts characters as the server does, an emoji as one
  ([GOV.UK character count](https://design-system.service.gov.uk/components/character-count/)).
- **No autofill where it would be wrong**: a workspace's name is not the
  person's name; a word to type exactly has no spelling marks
  ([WCAG 1.3.5](https://www.w3.org/WAI/WCAG22/Understanding/identify-input-purpose.html)).
- **Text spacing a person sets reaches the field.**

Not done yet, and why: dropping the browser's own required check in favour
of the component's error beside the field needs every form's server side to
say it first; a count shown only near the limit (the limits here are small).
