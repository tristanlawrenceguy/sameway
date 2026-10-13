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
- **A turn you can watch.** While the assistant works, the status beside
  Send says what it is doing in words, a few seconds apart, and the reply
  grows in short bursts with no caret; a screen reader hears the reply
  once, whole, when it is done. A block about to be added holds its place
  on the canvas in the assistant's colour
  ([WCAG 4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html);
  design/foundations/motion.md).
- **A long chat shows its latest twenty messages**, and a link above
  them that says how many came before ("34 earlier messages") opens the
  whole conversation at the first message shown before. It is a page
  with every message in it, not a list that loads as you scroll: a
  screen reader user of a chat that loads earlier messages on demand
  loses their place each time it shifts, and a list that keeps growing
  makes a place hard to find again
  ([NN/g on infinite scrolling](https://www.nngroup.com/articles/infinite-scrolling-tips/),
  [WCAG 2.4.4 link purpose](https://www.w3.org/WAI/WCAG22/Understanding/link-purpose-in-context.html)).
  An agent reading the page with look gets the same twenty and the same link.
- **What it changed is read from what ran, not from its words.** The
  changes under a reply come from the log. A reply that opens by saying
  it made something in a turn that changed nothing is asked again once;
  still saying so, it carries "(Sameway checked: nothing was changed.)",
  named as Sameway's so it does not read as the assistant taking its
  words back. The API, exports and look read the same words
  ([Microsoft guideline 11, make clear why the system did what it did](https://www.microsoft.com/en-us/research/blog/guidelines-for-human-ai-interaction-design/)).
- **What Sameway says to the model says it is Sameway's.** A provider
  takes notes to the model only in the person's turn; each begins "From
  Sameway, not the person", and a file's words come between markers as
  data, as a record's do
  ([OWASP LLM01](https://genai.owasp.org/llmrisk/llm01-prompt-injection/),
  [spotlighting](https://arxiv.org/abs/2403.14720)).
- **Its scripts are its own.** Pop out, the log's remembered height and Enter to send live in this component's enhance.js (they were design/base/15-chat-shape.js and 20-compose-enter.js). The turn shown as it happens stays in design/base (17-live.js, 34-turn.js): it reaches the canvas and the status, not only the chat.

Not done, and why: a confirmation before deleting (Undo does the same
without a question each time); sending on Enter on a phone (there is then
no way to make a new line); a role of log on the messages (screen
readers support it poorly; new messages are said through the status
line); loading earlier messages in place as you scroll (the place moves
under a screen reader, and the link already keeps it); a separate
"no changes" receipt under every reply (most replies are answers, and
a line under each would be noise: only a reply that claims a change
and made none says so); hiding a reply that claims what did not happen
(the person would not see what the model said, and the note already
says it was not done).
