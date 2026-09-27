# message

Use a message for one turn of a conversation. Put messages inside an ordered
list (`ol`) so assistive technology can announce how many turns there are and
move between them. The `error` role is for system notices such as a failed
model call, so problems appear in the transcript where they happened.

Each item of `changes` is an [event](../event/README.md), drawn compact:
the same line the activity log shows for that change, without the actor
the message's heading already says. The server builds both from the log
entry with one function, so give an item the event's props: `action`,
`target`, `detail`, `href`, and `undo` while it can still be undone, which
posts back to `from`. Give `undo` only under the newest reply: undo is a
moment, and the activity log keeps the control for everything older.

A setting change is sent in words, as the log says it:
`{"action": "changed", "target": "text size", "detail": "to Large"}`,
never `{"action": "set", "component": "ui.text"}`.

The optional `links` prop accepts an array of link objects with `href` (required)
and `label` (optional). Links render as inline pills between content and changes.
When omitted or null, links render silently without error.

## Why it works this way

- **Each turn starts with a heading**, who said it and when: most screen
  reader users move through a long page by its headings, and few can move
  by articles ([WebAIM survey](https://webaim.org/projects/screenreadersurvey10/)).
- **Who said it is a word**, with a dot, never colour or side alone; a
  reply that failed says "did not finish"
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **Line breaks kept**, so a list written line by line reads as one
  ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **What a reply changed is labelled where it can be seen**, one change to a
  row, each row 44px, so pressing Undo never lands on the change above
  ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).
- **Each change is an event, said as the log says it.** The list drew its
  own lines once, and every wording fix (a raw setting name, a "(note)"
  after a title) had to be made twice and was missed in one; now "changed
  text size to Large" is written in one place and reads the same under
  the reply, on /activity and in Undo
  ([WCAG 3.2.4](https://www.w3.org/WAI/WCAG22/Understanding/consistent-identification.html)).
- **A time with its moment**, and its day when not today, the same whether
  the page was loaded or the message arrived live.
- **New messages are said through the chat's status line**, not a log
  region, which screen readers support poorly
  ([ARIA issue on the log role](https://github.com/w3c/aria/issues/1104)).

Not done, and why: a copy button on every message (a tab stop per turn;
selecting and copying already works); avatars and left and right bubbles
(a weak cue zoomed in and in a narrow window); folding long replies (hides
words from find and from screen readers).

