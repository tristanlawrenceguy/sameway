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
