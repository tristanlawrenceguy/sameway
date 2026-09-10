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
  shows a **badge** such as "Added by assistant, edited by you · 20:24".
  The record behind it has `actor` and `created_by` fields.
- Every **event** in the activity log names the actor as a word and carries
  `data-actor`, `data-action`, and `data-target`.

## What changed: receipts and markers

- After an assistant turn, the reply carries a **receipt**: a list of the
  canvas changes it made ("added heading Shopping", "removed list"). It is
  stored on the message record as `changes` and rendered as
  `.sw-message__changes` with `data-action` and `data-target` per item.
- Blocks touched in the last turn get `data-changed="added"` or
  `"updated"`. They flash once in the actor's colour, and a visually hidden
  note ("added in the last turn") is read by screen readers.
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

## For agents

Query by attribute, act by role and name:

```
[data-component=status][data-state]        is anything in flight, how did it end
[data-block-id][data-changed]              what changed in the last turn
[data-block-id][data-actor=human]          blocks a person has touched
[data-component=message][data-role=error]  failures, in the transcript
.sw-message__changes li[data-action]       the receipt for a reply
GET /api/activity                          the full log, newest first
```
