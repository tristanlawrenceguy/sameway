# person

Someone, by name, with the colour they have everywhere in the workspace:
who a task is for, who else is here, who changed something. The same
person has the same colour on every computer that hosts the workspace,
worked out from their Tailscale login.

## The name carries it

The dot helps a person who sees colours tell people apart at a glance, but
nothing depends on it: the name is always there, in the text colour, at
full contrast, and the dot is hidden from screen readers. Six colours are
few enough to tell apart; two people may share one, and their names still
say who is who. In forced colours every dot is the same, and the names
still say who.

## What a screen reader hears

The label and the name, once: "For Hana". Nothing of the dot, and no
"avatar" or "image".

## Names as people write them

Give the name in full, as the person writes it. It is never cut short: a
long name wraps onto the next line, and a name with no spaces, such as a
login standing in for one, breaks where it must. A name in Arabic or
Hebrew keeps its own direction and does not reorder the words or commas
beside it.

## Not a link

A person chip names someone; it does not lead to their page. Where it
should, put a link beside it.

## Why it works this way

- **Colour is a second way of saying it.** Colour alone would leave out
  everyone who cannot see it or tell the six apart, so the name always
  says who
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **The dot is decoration.** A mark beside a name that says the same thing
  is hidden, so the name is not heard twice. It is also why the dot need
  not meet 3:1, though the six colours do anyway
  ([WCAG 1.4.11](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html),
  [Atlassian avatars](https://aui.atlassian.com/aui/latest/docs/avatars.html)).
- **The name is isolated.** It sits in a `bdi`, so a right-to-left name
  among left-to-right words, or the other way round, keeps the punctuation
  after it in place
  ([W3C inline bidi markup](https://www.w3.org/International/articles/inline-bidi-markup/)).
- **Long names wrap.** The chip flows like text and may break anywhere, so
  nothing runs off a 320px screen
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html),
  [GOV.UK names](https://design-system.service.gov.uk/patterns/names/)).
- **No language on the name.** A person's name is a proper name, which
  needs no `lang` of its own
  ([WCAG 3.1.2](https://www.w3.org/WAI/WCAG22/Understanding/language-of-parts.html)).

Not done, and why: initials in a coloured circle (they say the name again
for those who see it, and are wrong for one-letter names, names with no
first and last, and many scripts, per
[W3C personal names](https://www.w3.org/International/questions/qa-personal-names));
photos (a picture of someone is theirs to share, and the name already says
who); cutting a long name short with an ellipsis (the end of a name is
often what tells two people apart); a colour each person chooses, or one
kept apart from everyone else's (the colour comes from the login alone so
every computer agrees without asking another).
