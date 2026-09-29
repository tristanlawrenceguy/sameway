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
`as: cards` puts them in cards, each pressed anywhere to open it.
`as: board` puts a column for each choice of a pick-list field, named
with `by` (status: To do, Doing, Done), in the order the field lists
them, each saying how many it holds, and a column for those with none.
A line over the board names every column with its count and leads to it.
Each card has a Move form: its column picked from a list and a Move
button (the move component), which saves the field like any edit, comes
back to that button focused in the card's new column, and says "House moved from Active to Done." with an Undo. A list
of one line per record is a list; use a table when the fields are the point
and several are compared.

`limit` is how many are shown, 20 unless given. When more match, the list
says it shows the first of them, and its link, See all of them, leads to
the list page with the same query, which has them all.

The person looking at it can narrow and sort it where it is, without
asking. Over the list are a few choices made from the type's fields: Sort
(Newest first, Oldest first, A to Z, and a date field soonest first), and
one each of a pick-list (Status: Any, Active, Done), a yes-or-no (Done:
All, Not done, Done) and a date field (Due: Any time, before today, in the
next 7 days), where the type has them. A field `where` already fixes is not
offered, nor is a board's own column field. Apply shows them; under the
form the list says how many match and, once, what it is showing: "2 tasks.
Showing: not done, due soonest first." with Reset beside it, back to how it
was set up. The choices only add to `where`, never take from it: the
assistant decides what the list is about, the person narrows within it.

They are the [filters](../filters/README.md) component as a form, a sort
and up to three fields chosen together, the same as every list that
narrows. They are a plain GET form: the choices are in the page's address, named
after the block (`?c-<block>-sort=due&c-<block>-done=false`), so two lists
on a canvas keep their own, the form keeps the page's other fields, a live
refresh and a reload keep them, the link to the list page carries them,
and none of it needs a script. They are offered on the block's own page,
and on the canvas at full size when more than five match and `limit` is
over five; `controls: false` takes them away, for a short list or a small
space, and `controls: true` offers them anyway. Never in a side pane.

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

- **A board without dragging.** A card moves column when its field
  changes, on its page or with its press, so a keyboard, a switch and the
  assistant move it the same way
  ([WCAG 2.5.7](https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html)).
  Columns are sections with headings, read in order, and stack on a phone.
  The Move form is a native select and button: no script, and a voice can
  say "click Move" because the button's name starts with what it shows
  ([GitHub's testing of a move form](https://github.blog/engineering/user-experience/exploring-the-challenges-in-creating-an-accessible-sortable-list-drag-and-drop/),
  [WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html)).
- **Back where you were, told what happened.** After a move the page
  returns to the card, not its top, and the message names the card and
  both columns with an Undo
  ([W3C COGA, make it easy to undo](https://www.w3.org/TR/coga-usable/)).
- **A tally for those who cannot jump by heading.** A keyboard or switch
  moves by Tab, not headings, so the line over the board leads to each
  column.
- **No sideways scrolling, no tabs or swiping on a phone.** Columns stack,
  as reflow asks, and nothing is hidden behind a gesture
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html),
  [NN/g on carousels](https://www.nngroup.com/articles/mobile-carousels/)).
- **Still a list to Safari.** Plain lists use an empty marker rather than
  none, so VoiceOver keeps "list, 3 items"
  ([Scott O'Hara](https://www.scottohara.me/blog/2019/01/12/lists-and-safari.html)).

- **Narrowed where it is, on the server.** A person should not have to
  ask for "only the overdue ones"; a GET form with a button works with no
  script, and the address says what is shown, so it can be kept and sent
  ([GOV.UK finder-frontend](https://github.com/alphagov/finder-frontend),
  [MoJ filter a list](https://design-patterns.service.justice.gov.uk/patterns/filter-a-list/)).
- **Apply, not on change.** Choosing an option changes nothing until
  Apply: a new page on every change moves the ground under a keyboard and
  a screen reader, and a person setting two choices waits for one
  ([WCAG 3.2.2](https://www.w3.org/WAI/WCAG22/Understanding/on-input.html),
  [W3C H32](https://www.w3.org/WAI/WCAG22/Techniques/html/H32),
  [DWP research on filters](https://design-system.dwp.gov.uk/research/filters),
  [NN/g on batch and interactive filters](https://www.nngroup.com/articles/applying-filters/)).
- **What is shown, said once, with a way back.** The choices in words,
  not a count of them, next to the results, and one Reset
  ([Baymard on applied filters](https://baymard.com/blog/how-to-design-applied-filters),
  [DWP research on filters](https://design-system.dwp.gov.uk/research/filters)).
  How many match is said in words; nothing matching says the conditions
  and Reset is beside it.
- **A few choices, in plain words**: at most a sort and three fields,
  each choice a short phrase, Any or All first
  ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **The sort said in words, not by aria-sort**, which belongs on a
  table's own column headers when they sort it; a list's order is said
  in the line over it, where every person reads it
  ([Adrian Roselli on sortable table columns](https://adrianroselli.com/2021/04/sortable-table-columns.html)).
- **Narrow, never widen.** The assistant's `where` is what the list is
  for; the address can only add what the form offers, so a link cannot
  make a list show what it was built to leave out.
- **Nothing last.** Sorted by a date, those with no date come after the
  ones with one, so "soonest first" starts with what is soonest.

Not done, and why: radios for the choices (the research prefers them to
dropdowns, but four groups of them would be taller than many lists; each
dropdown here has three or four short options, labelled in sight);
submitting on change (3.2.2, above); counts beside each option (each would
be a query per option on every render, for a handful of records); choices
kept between visits (the address keeps them, which is enough; MoJ advises
filters not persist unless people need it); dragging cards between columns (the one way that
leaves out a keyboard unless a second way is built beside it); field names only for screen readers (sighted people
need them as much); a table squashed to fit a phone (its columns become
unreadable, so it scrolls in a box instead).
