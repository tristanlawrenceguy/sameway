# badge

Use a badge to say who did something ("Assistant") or what state a thing is
in ("Draft", "Failed"). Keep it to what is not already obvious: no
timestamps, and never name the same actor twice. The label carries the
meaning; the tone only colours it.

## Not a control

People take badges for buttons when they look like one, and try to press
them. So a badge has no border, no hover and no pointer, and:

- Word it as a state, never a verb: "Published", "Needs review", not
  "Publish" or "Review".
- Keep it out of links and buttons, and off their edges, where it reads as
  part of them.

## Heard out of place

A screen reader reads a row of badges one after another: "Done, Pinned,
High". Where the label alone does not say what it is, give `context`, read
after it and not shown: "High" with context "priority" is heard as "High
priority".

## Tones

Provenance tones (human, assistant, system) are only for who did something:
they also set `data-actor`, the attribute the rest of the system reads to
tell people's actions from the model's. A day, a count or a name is not
provenance: use info or neutral. Add a tone only when a new state needs
telling apart from the others; fewer tones are easier to learn.

## Long labels

A long label, such as a record's name, wraps inside the badge instead of
running off a narrow screen, and the dot stays on the first line. In forced
colours the tint goes, and the dot is drawn in the text's colour.

## Why it works this way

- **Never a control.** No border, hover or pointer, worded as a state, and
  kept out of links and buttons, because people take a tag that looks like
  a button for one and press it
  ([GOV.UK tag](https://design-system.service.gov.uk/components/tag/),
  [USWDS tag](https://designsystem.digital.gov/components/tag/)).
- **Heard in context.** `context` is read after the label, so "High" in a
  row of badges is heard as "High priority"
  ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **Wraps.** A long label wraps in its box instead of running off a narrow
  screen ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html)).
- **Keeps its dot in forced colours**, drawn in the text's colour, so a
  state still reads as one when the tints go.
- **Provenance tones only for who did something**, since they set
  `data-actor`, which the rest of the system reads.

Not done, and why: a border in forced colours (it made badges look like
buttons); a time in a badge (it is not a state, and says what the row
already says). A record's day under its title is the one
exception, being where it stands rather than when something happened:
"Due Fri 9 Oct", "Overdue, due yesterday", named by its field and held in
a time element around the badge (design/foundations/glance.md).
