# State language

People and the assistant share one canvas, and things happen while you
wait. The system therefore has a small vocabulary for *who*, *what changed*,
and *what is happening*, and it always speaks it three ways at once: as
text, as colour, and as attributes.

## Who did it: provenance

| Actor | Text | Colour | Attribute |
|---|---|---|---|
| A person | "You" | human indigo | `data-actor="human"` |
| The assistant | "Assistant" | assistant teal | `data-actor="assistant"` |
| The software | "System" | system amber | `data-actor="system"` |

Where it appears:

- Every **message** carries `data-role` (user, assistant, error) and
  `data-actor`. The author word is the first thing in the article.
- Every **canvas block** carries `data-actor` (who last changed it) and
  shows a **badge** such as "Assistant, edited by you", with the time beside it rather than in it.
  The record behind it has `actor` and `created_by` fields.
- Every **event** in the activity log names the actor as a word and carries
  `data-actor`, `data-action`, and `data-target`.

## What changed: receipts and markers

- After an assistant turn, the reply carries a **receipt**: a list of the
  canvas changes it made ("added heading Shopping", "removed list"). It is
  stored on the message record as `changes` and rendered as
  `.sw-message__changes`, one event (`data-component="event"`, compact) per
  item with `data-action` and `data-target`: the same line the activity log
  shows for that change.
- Blocks touched in the last turn get `data-changed="added"` or
  `"updated"`. They flash once in the actor's colour, and a visually hidden
  note ("added in the last turn") is read by screen readers.
- Those blocks also get `data-arrival="1"`, `"2"`, ... in the order they
  changed, and arrive on the page in that order, one after another, each in
  three stages (see motion.md, Arrival). The conversation block never
  arrives; it is the person's own tool.
- Everything, by anyone, lands in the **activity** content type:
  `/activity`, `/api/activity`, `sameway activity list`.

## What is happening: status

The **status** component is a live region with `data-state`:

| State | Meaning | Who sets it |
|---|---|---|
| `idle` | Nothing in flight | Server, on a fresh page |
| `working` | A request is in flight | `status/enhance.js` on submit |
| `done` | The last request succeeded; text says what changed | Server |
| `error` | The last request failed; text says so, `aria-live="assertive"` | Server |

While working, the form has `aria-busy="true"`, its submit buttons are
`aria-disabled`, the enclosing `[data-region]` has `data-state="working"`,
and the tab title gains a prefix. Without JavaScript the page still reloads
with the outcome; the enhancement only makes the wait visible.

## A state kept is true at any age

A status line is live: it says what is happening while it happens. A
value kept on a record is not: it is read days later, so its name says
what happened, not what is happening. A reminder that rang is "Rang",
not "Ringing" (the clock says what is ringing now). A name that says
something is under way ("Doing", "Waiting", "Being read") is only for a
value something here ends, and `internal/schema/states_said_test.go`
lists each with what ends it: the person, the clock, or Sameway. A file
left "Being read" when Sameway closed is put right when it opens
(`server/media/left_over.go`): a recording waits to be written down
again, any other file says it was not read and to add it again.

## Words that are the system's own

Nothing a person reads is the system talking about itself:

- **No sizes too small to decide on.** A size is said so people can
  decide on it. A download says its size from 1 MB (`blocks.DownloadSize`);
  a size is MB from 1 MB, with a tenth below 10 MB ("2.1 MB"), KB below,
  and never a count of bytes ("under 1 KB").
- **No words as code writes them.** Enum values as stored, field names,
  settings keys (`meetings.teams_client_id`), error codes and JSON Schema
  keywords stay with agents. A page asks for what a setting holds in
  words ("give the assistant its client id to keep").

The page crawl, `internal/server/machine_words_test.go` with
`plain_words_crawl_test.go`, reads every page a lived-in workspace serves
and fails on each class, so the next instance is found there rather than
by the crew one page at a time.

### Why it works this way

- NN/g's first heuristic asks that status be current: a status shown
  after it stopped being true misleads
  ([visibility of system status](https://www.nngroup.com/articles/visibility-system-status/)).
- NN/g's second asks for the person's words over the system's
  ([ten heuristics](https://www.nngroup.com/articles/ten-usability-heuristics/)).
- GOV.UK writes file sizes in KB below 1 MB and MB above, to the tenth or
  hundredth ([A to Z, numbers](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/style-guides/a-to-z-style-guide/)).
  Sameway leaves the small ones out on a download (crew finding 0711):
  every record's file is small, and a size that never differs says
  nothing.

Not done, and why: a list of banned words. What counts as the system's
own differs by workspace (its fields, its values, its ids), so the crawl
reads them from the workspace and from the shape of the word.

## For agents

Query by attribute, act by role and name:

```
[data-component=status][data-state]        is anything in flight, how did it end
[data-block-id][data-changed]              what changed in the last turn
[data-block-id][data-arrival]              its place in the order changes arrive
[data-block-id][data-actor=human]          blocks a person has touched
[data-component=message][data-role=error]  failures, in the transcript
.sw-message__changes [data-action]         the receipt for a reply, as events
GET /api/activity                          the full log, newest first
```

## Drafts

Words typed and not yet sent or saved are not lost by leaving the page.
The chat composer keeps its draft per chat, and an inline edit keeps its
fields, in the browser under the workspace's origin
(`sameway:draft:...` in localStorage). A draft is restored when the same
composer or edit is opened again: the composer fills itself, and a block
with an unfinished edit says so at the top with Continue editing and
Discard. A draft goes the moment the message is sent or the edit is
saved, or cancelled on purpose. Nothing about a draft is sent to the
server; see `design/base/18-drafts.js`.
