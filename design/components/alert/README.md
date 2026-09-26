# alert

A short message that changes what the person does next: a save that worked,
a problem to fix, a request that failed. Keep it to one or two sentences.

## Words

Say what happened, then what to do, in the person's words: "Not saved. The
title is needed." rather than an error code. Do not blame the person, and do
not pass on a program's raw error text; say it plainly or leave it out.

## Kind

The kind is said four ways, so nobody depends on colour (WCAG 1.4.1, a
non-colour cue): a word a screen reader says first ("Error:", "Warning:",
"Success:", "Information:"), a mark of its own shape (ℹ information, ✓
success, ⚠ warning, a filled circle with ! for an error), a coloured edge,
and a tinted ground. In forced colours the tints go and the edge of a warning
or an error turns double. The mark is hidden from screen readers, which hear
the word instead of "warning sign". An `icon` prop replaces the mark when a
different one suits the message better.

## When it is said

A message already on the page when it loads is not announced by screen
readers, whatever its role; it is found where it sits, under its heading.
Set `live` only for a message put on the page after it loaded, or for the
outcome of an action, where focus is also moved to it: `live` gives it
`role=alert` (warning, danger) or `role=status` (info, success).

## Title

A `title` is a heading (level 2 unless `level` says 3 or 4), so a person
moving by headings finds a page's problem. Leave it out for a one-line
message.

## Closing

`dismiss` adds a close button, for messages that are over once read, such as
the outcome of an action. Keep a problem the person still has to fix on the
page until it is fixed. Nothing closes itself.

Use alerts sparingly: people learn to skip boxes that appear often. A problem
with one field belongs next to that field, or in an error summary.

## Why it works this way

- **Its kind in words.** A word, a mark, an edge and a tint each say the
  kind, and a screen reader hears the word first, so nobody depends on
  colour ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html),
  [GOV.UK warning text](https://design-system.service.gov.uk/components/warning-text/)).
- **A mark for each kind.** Warning and error have marks of different
  shapes, and the mark is hidden from screen readers, which would say
  "warning sign" before the word
  ([USWDS alert](https://designsystem.digital.gov/components/alert/)).
- **A heading for its title**, so a page's problem is found by moving from
  heading to heading
  ([WCAG 1.3.1](https://www.w3.org/WAI/WCAG22/Understanding/info-and-relationships.html)).
- **A live role only where it is heard.** A message on the page when it
  loads is not announced whatever its role, so only a message that arrives
  later, or the outcome of an action, gets one
  ([APG alert pattern](https://www.w3.org/WAI/ARIA/apg/patterns/alert/),
  [Inclusive Components notifications](https://inclusive-components.design/notifications/)).
- **Close is a 44px target named Close message**, in the corner, clear of
  the words ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).
- **Plain words, no raw errors.** Say what happened and what to do
  ([NN/g error messages](https://www.nngroup.com/articles/error-message-guidelines/)).

Not done, and why: messages that close themselves (a person who reads
slowly or looked away misses them); `role=alert` on every message (it is
not heard on page load, and heard too often it is noise); one mark shared
by warning and error (the shape then says nothing).
