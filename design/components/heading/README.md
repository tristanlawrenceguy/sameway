# heading

Use a heading to start a section. The page layout already provides the h1, so
content added to a page starts at level 2. Do not pick a level for its size;
pick it for its place in the outline and let the stylesheet size it.

## Why it works this way

- **Level 2 to 6 only.** Every page has one h1, which the layout gives, even
  when it is hidden on a canvas; a heading block at level 1 made a second.
  ([WAI headings tutorial](https://www.w3.org/WAI/tutorials/page-structure/headings/))
- **One scale.** A heading block's h2 and a text block's `##` are the same
  size, so what the page shows agrees with its outline
  ([WCAG 1.3.1](https://www.w3.org/WAI/WCAG22/Understanding/info-and-relationships.html),
  [GOV.UK headings](https://design-system.service.gov.uk/styles/headings/)).
- **Close to what it introduces.** On a canvas a heading block sits nearer
  the block under it than the one above
  ([USWDS typography](https://designsystem.digital.gov/components/typography/)).
- **Never longer than a line can be read**, balanced over its lines
  ([WCAG 1.4.8](https://www.w3.org/WAI/WCAG22/Understanding/visual-presentation.html)).
- **The whole title.** A record's page heading is its whole title, wrapped
  as it needs; only one-line places (the window title, rows, crumbs) cut it,
  at a word, with an ellipsis
  ([WCAG 2.4.6](https://www.w3.org/WAI/WCAG22/Understanding/headings-and-labels.html)).
- **An id that works as an anchor**: lowercase words joined by hyphens.

Not done, and why: a size chosen apart from the level (it invites picking a
level for its look); a # link beside every heading (a tab stop each, clutter
on a canvas of headings); ids made from the words (they change when the
words do); a hard length limit (it would stop the assistant rather than
inform it).
