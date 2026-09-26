# textarea

Use a textarea for more than one line of input: a message, a note body, a
description. Enter inserts a newline; the form's submit button sends. In the
chat's message box, on a device with a keyboard, Enter sends and Shift+Enter
starts a new line, and the hint says so; on a touch screen, or without
scripts, Enter makes a new line. Shares
its stylesheet classes with `text-field`.

Set rows to the length you expect: 3 for a message, 5 to 8 for a
description. The box grows as they type, so a small box is not a limit.

## Why it works this way

- **It grows with what is typed**, so a long message is seen whole, not
  scrolled in a small box, and stops at most of a screen so Send stays in
  reach ([NN/g on form design](https://www.nngroup.com/articles/web-form-design/),
  [MDN field-sizing](https://developer.mozilla.org/en-US/docs/Web/CSS/field-sizing)).
- **Its first size says what to expect**: a few lines for a message, more
  for a description ([GOV.UK textarea](https://design-system.service.gov.uk/components/textarea/)).
- **Enter that finishes a word in Japanese, Chinese or Korean does not
  send** the half-written message, and an empty box sends nothing.
- **A blank first line is kept.**
- **Counted as the server counts**: a line break is one character.

Not done, and why: a script that resizes from the text's height (it
flickers on paste, and rows already hold without field-sizing); a word
count (the limits here are in characters).
