# message

Use a message for one turn of a conversation. Put messages inside an ordered
list (`ol`) so assistive technology can announce how many turns there are and
move between them. The `error` role is for system notices such as a failed
model call, so problems appear in the transcript where they happened.

A change in the receipt that can still be undone carries `activity`, the id
of its log entry, and the receipt shows an Undo form for it, posting back to
`from`. Give it only under the newest reply: undo is a moment, and the
activity log keeps the control for everything older.

The optional `links` prop accepts an array of link objects with `href` (required)
and `label` (optional). Links render as inline pills between content and changes.
When omitted or null, links render silently without error.
