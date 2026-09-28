# voice

The microphone, in one button, in two ways.

**Record** sits beside a file field (the upload component carries it, and
the chat's Attach a file). Press Record, speak, press Stop recording: the
recording goes into the field, named for when it was made, and is added
by the form's own Upload or Send. A recording added this way is written
down like any other.

**Dictate** sits by the chat's message once speech-to-text is on the
computer that hosts the workspace. Press Dictate, speak, press Stop
dictating: what was said is written down there and put into the message
where the cursor was, to be read and changed before it is sent.

It is shown only where the browser can record: a secure page (https, as
Tailscale gives, or this computer) and a browser with a microphone. The
microphone is asked for when the button is pressed, never before.

## Why it works this way

- **Nothing sent by itself.** A recording waits in its field and dictated
  words wait in the message, so a person hears or reads what they are
  about to send ([W3C COGA, let users check](https://www.w3.org/TR/coga-usable/)).
- **Words on the button, a quiet timer.** Record changes to Stop recording
  (a voice can say what it sees, [WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html));
  the time is a timer, which a screen reader does not read on every tick
  ([MDN on the timer role](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/timer_role)),
  and the status says each change once.
- **Asked for when pressed.** A permission asked as a page opens is
  refused without being read; a refusal is said with what still works.
- **Written down at home.** Dictation uses the same speech-to-text as
  recordings, on the computer that hosts the workspace; nothing is sent
  elsewhere. The operating system's own dictation (Voice Access, macOS and
  iOS dictation) types into the message too, and nothing here stands in
  its way.

Not done, and why: dictating into every field (the operating system does
that already, everywhere); sending as soon as dictation ends (the words
are read first).
