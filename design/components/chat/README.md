# chat

Use chat for a conversation: a named region holding a transcript and a
composer. It is an ordinary component, so on a Sameway canvas it is a block
like any other. It can be moved, widened, restyled with `layout`, or removed
entirely, and the assistant can put it back.

Because it can be removed, the conversation is never only here. The same
transcript and composer are always at `/chat`, and agents can post to
`/api/chat`, so nobody is locked out by a layout choice.

Use `layout: bare` when the region already sits inside a surface and the
panel chrome would double up. The title stays in the accessibility tree
either way.

## Why it works this way

- **Delete and clear can be taken back.** Delete chat and Clear
  conversation keep what they remove in the log and offer Undo on the
  page, as other changes do, since one press should not lose a whole chat
  ([WCAG 3.3.6](https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-all.html)).
- **Things to ask.** An empty chat offers a few, each putting its words in
  the box to read and send, so a blank box is not the only way in. They go
  once the conversation starts.
- **Keyboard scroll.** The transcript, named Messages, takes Tab once, so a
  keyboard can scroll it when it holds no links
  ([Adrian Roselli on keyboard scrolling areas](https://adrianroselli.com/2022/06/keyboard-only-scrolling-areas.html),
  [WCAG 2.1.1](https://www.w3.org/WAI/WCAG22/Understanding/keyboard.html)).
- **Enter makes a new line on a touch screen**, whose keyboard has no
  Shift+Enter; the hint says Enter sends only where it does
  ([MDN enterkeyhint](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes/enterkeyhint)).
- **A failure said where it is.** A failed turn's status says the last
  request failed and why, never "see below"
  ([WCAG 1.3.3](https://www.w3.org/WAI/WCAG22/Understanding/sensory-characteristics.html)).
- **A message from another day says its day** with its time.

Not done, and why: a confirmation before deleting (Undo does the same
without a question each time); sending on Enter on a phone (there is then
no way to make a new line).
