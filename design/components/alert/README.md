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
