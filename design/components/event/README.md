# event

Use an event for one line of activity: "Assistant added table", "You removed
list", "System failed: could not reach the model". The actor is always
written as a word, and `data-actor`, `data-action`, and `data-target` carry
the same facts for machines. Put events in a list; the page decides whether
that list is a receipt under a reply or the full activity log.

An entry that can still be undone is given `undo`, the address its Undo
form posts to, and `from`, the page to return to. The control is a real
form, so it works without JavaScript and reaches assistive technology as a
button named Undo. Leave `undo` out once the thing is no longer as the entry
left it; the log, not the entry, decides that.

In a log read heading by heading, give `level`: the entry's sentence is then
the heading, said once, with its time and Undo beside it, not a heading with
the same sentence under it. Give `datetime` for the time element, and put the
day in `time` wherever no heading above says it (2 Jan 14:05); on a page
grouped by day, the time alone. `via` says where it was done from when not
this computer (on pixel-7, through the command line).

Undo says what it undid: "Undone. You added card Plan."
