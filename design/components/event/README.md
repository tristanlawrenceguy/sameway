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

## Why it works this way

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
