# talk

Voice mode: a spoken conversation with the assistant, in the chat.

Press Start voice mode and speak. When you stop for a moment (or press
Done speaking), what you said is written down on the computer that hosts
the workspace, put in the message and sent. The reply is written in the
chat as always and read aloud in a voice on this device, and then it
listens again. Skip the reply stops the reading; End voice mode, or
Escape, stops everything and turns the microphone off.

It is shown where the browser can record (a secure page) and
speech-to-text is on the host. Replies are read only in a voice that
runs on this device; where there is none, it says so and the replies are
shown.

## Why it works this way

- **Everything is written.** What was said and every answer appear in the
  chat as they happen, so nothing is only heard
  ([WCAG 1.2](https://www.w3.org/WAI/WCAG22/Understanding/media-equivalents-audio-description.html)).
- **The microphone is never left open.** It is asked for when pressed,
  not listened to while a reply is read (so the assistant does not hear
  itself), paused after 30 seconds with nothing heard, and stopped at once
  by End voice mode or Escape.
- **State in words.** Listening, Hearing you, Writing down what you said,
  The assistant is answering, Reading the reply: said once as it changes,
  and shown; the red mark beside Listening is a second sign, not the only
  one, and nothing moves.
- **Done speaking**, for anyone whose pauses are longer than a second, so
  a turn is never cut off by a timer they cannot control
  ([WCAG 2.2.1](https://www.w3.org/WAI/WCAG22/Understanding/timing-adjustable.html)).
- **Read replies aloud can be turned off**, and stays off on this device:
  a screen reader already reads the chat's replies, and hearing each twice
  helps nobody.
- **Voices on this device only.** A browser's online voices send the text
  away to be spoken; those are not used.
- **Written down at home**, with the same speech-to-text as recordings and
  dictation; the sound is not kept.

Not done yet, and why: speaking over the reply to interrupt it (the
microphone would hear the reply itself; Skip the reply does it by a
press); a voice fetched to this computer where the device has none.
