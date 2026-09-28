# fields

A thing's facts, each with its name: Status Draft, Due Fri 2 Oct, Project
Garden. A record's page shows what its heading and chips have not already
said this way.

A value can be words, a link to what it names (`href`), or structured text
(`markdown`), which keeps its headings and lists. Give a value its `prop`
(and its stored `source`, `kind` or `options` when they differ from what
reads) and the inline editor knows which field it is; without them it is only
read.

Set `compact` for facts inside a card, a row or a record block, where they
are a detail rather than the page: smaller and closer, without hairlines.
Components that show facts (record, collection) render this component
through the `fields` template helper rather than a list of their own.

Leave empty facts out rather than showing a name with nothing beside it.

A fact names itself as the record's own page names it (a field's label), and
a fact that points at another record leads to it, wherever it is shown.

## Why it works this way

- **Name beside value, then above it.** Each fact is a row, name and value
  side by side while the value keeps most of the width, and the name moves
  above as soon as it would not, so a long name never squeezes its value to
  a letter a line in a card or on a phone. The row is a `div` inside the
  `dl`, which HTML allows and screen readers ignore.
  ([GOV.UK summary list](https://design-system.service.gov.uk/components/summary-list/),
  [the dl element](https://html.spec.whatwg.org/multipage/grouping-content.html#the-dl-element),
  [Adrian Roselli on description lists](https://adrianroselli.com/2022/12/brief-note-on-description-list-support.html))
- **No blank facts.** A name with nothing beside it is left out, not shown
  as an empty row or a dash that a screen reader reads out; adding a value
  is the editor's job.
- **The same name everywhere.** A field is called what the schema calls it
  on its page, in a card and in a table
  ([WCAG 3.2.4 Consistent Identification](https://www.w3.org/WAI/WCAG22/Understanding/consistent-identification)).
- **Line breaks kept**, so a list's lines and a note's paragraphs read as
  written.
- **A link alone in a value is a 44px target**, reaching into its row's
  padding ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).

Not done, and why: a Change link on every row (it repeats the block's one
Edit, and is a lot to read past on a phone); ARIA term and definition roles
(they break list counts in Safari with VoiceOver); a fixed label column
(wasted width for short names in cards); upper-case names (harder to read).
