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
