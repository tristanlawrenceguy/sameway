# event

Use an event for one line of activity: "Assistant added table", "You removed
list", "System failed: could not reach the model". The actor is written
as a word (a compact line leaves it to its message), and `data-actor`,
`data-action`, and `data-target` carry the same facts for machines. Put events in a list; the page decides whether
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
this computer, such as on a phone or tablet.

Undo says what it undid: "Undone. You added card Plan."

The full log, at `/activity`, can be narrowed by who made a change (You,
the assistant, the system, or another person by name), what it was to
(tasks, cards, settings: only kinds the log has) and when (Today, Last 7
days), chosen together and applied with Apply: the
[filters](../filters/README.md) component as a form. The choices are in
the address (`?who=assistant&kind=task&when=week`), the log says how many
match and what it shows with Reset, it goes a page at a time of 200 with
the pages keeping the choices, and an entry's `from` is the narrowed
address, so Undo comes back to the same narrowing.

Under a message that already says who, as its Changes made list, give
`compact`: the line starts at its verb ("changed text size to Large"),
with no actor and no mark, one 44px row, and no heading, since the
message has one. The message draws each change through this component,
and the server builds the props for that list and for the activity log
with one function, so a change reads the same in both.

## Why it works this way

- **One line for a change, wherever it shows.** The chat once drew its own
  receipt lines, and wording fixes landed in one place and not the other;
  now the receipt is these events, compact, and the actor it leaves out
  is the one its message's heading says, so it is not heard twice
  ([WCAG 3.2.4](https://www.w3.org/WAI/WCAG22/Understanding/consistent-identification.html)).
- **Each entry said once.** A heading with the same sentence under it was
  seen and heard twice, and the two could differ, so with `level` the
  sentence is the heading
  ([WCAG H69](https://www.w3.org/WAI/WCAG22/Techniques/html/H69),
  [MoJ timeline](https://design-patterns.service.justice.gov.uk/components/timeline/)).
- **Its time with its day.** A bare 14:05 from last week reads as today,
  so outside a page grouped by day the time carries its day, and every
  time is a time element with its moment
  ([MDN time element](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/time),
  [GOV.UK style guide](https://www.gov.uk/guidance/style-guide/a-to-z)).
- **The actor as a word**, so who did it never depends on a colour or a
  mark ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **Undo is a real form** and says what it undid, so the person knows the
  right thing was taken back
  ([NN/g user control and freedom](https://www.nngroup.com/articles/user-control-and-freedom/)).
- **Where it was done from**, when not this computer, so a change made
  elsewhere is not a surprise ([W3C COGA](https://www.w3.org/TR/coga-usable/)).

Not done, and why: times said only as "3 hours ago" (they go stale on a
page left open, and need a date to check); an Undo on an entry whose thing
has changed since (the log decides, so it never undoes the wrong state).
