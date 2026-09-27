# collection

Use a collection to show the records that match a query, kept current as
they change: the tasks due this week, the notes tagged garden, the drafts.
Give it `type` and, usually, `where`, `order` and a `label` that says what
the list is. The server fills in the matching records when the page
renders, so the list on the canvas and the list at `/t/<type>` with the
same query are one thing.

`where` is a list of conditions that must all hold, each `field`,
operator, value with no spaces: `done=false`, `status!=published`,
`title~garden` (contains), `tags=health` (has), `due<today`, `due<=+7d`,
`notes=` (empty), `due!=` (set). Dates take `2026-10-01`, `today`,
`tomorrow`, `yesterday`, `now`, `+7d`, `-1w`, `+3h`. `order` is a field, or
`-field` for the largest or newest first; `created_at` and `updated_at`
work too. The same words work on the list page as `?where=…&order=…`, in
the assistant's `find_records`, and as `--where` on the command line.

`brief` shows each title as a link with its state or date; `full` adds the
text. Nothing matching says so with the conditions; a wrong field says what
the type has instead, so the block never renders as nothing.

`show` names fields to show beside each title, with their names in sight:
"Due 19 Sep", not a date that could be any of them. `as: table` puts them
in columns, which scroll sideways on a phone in a box a keyboard can reach;
`as: cards` puts them in cards, each pressed anywhere to open it. A list
of one line per record is a list; use a table when the fields are the point
and several are compared.

`limit` is how many are shown, 20 unless given. When more match, the list
says it shows the first of them, and its link, See all of them, leads to
the list page with the same query, which has them all.

## Why it works this way

- **Its own name.** A list's heading id comes from its block, so two lists
  of one type on a canvas are each named by their own heading
  ([WCAG 4.1.2](https://www.w3.org/WAI/WCAG22/Understanding/name-role-value.html)).
- **Fields named in sight.** A date beside a title could be any date, so
  its field's name is shown, not only said to a screen reader
  ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **A table a keyboard can scroll.** On a phone it is a named region that
  takes focus
  ([Adrian Roselli on responsive tables](https://adrianroselli.com/2020/11/under-engineered-responsive-tables.html),
  [WCAG 2.1.1](https://www.w3.org/WAI/WCAG22/Understanding/keyboard.html)).
- **A list that keeps its marks**, the list component's own (`sw-list`):
  the rows once had them taken away, and Safari with VoiceOver then does
  not say it is a list or how many are in it
  ([Scott O'Hara on list-style none](https://www.scottohara.me/blog/2019/01/12/lists-and-safari.html)).
- **Cards that act as cards**, pressed anywhere to open, as elsewhere.
- **Says when it is cut short**, so records past the limit are not missed
  without a word.
- **Conditions read as a sentence**: "Nothing matches: not done and due
  before today."
  ([MoJ filter a list](https://design-patterns.service.justice.gov.uk/patterns/filter-a-list/),
  [NN/g empty states](https://www.nngroup.com/articles/empty-state-interface-design/)).

Not done, and why: field names only for screen readers (sighted people
need them as much); a table squashed to fit a phone (its columns become
unreadable, so it scrolls in a box instead).
