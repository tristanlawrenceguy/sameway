# talk

Voice mode: a spoken conversation with the assistant, in the chat.

Press Start voice mode and speak. When you stop for a moment (or press
Done speaking), what you said is written down on the computer that hosts
the workspace, said back to you, put in the message and sent. The reply is
written in the chat as always and read aloud in a voice on this device,
and then it listens again. Skip the reply stops the reading. When replies
are not read aloud, it lets the microphone go and waits for Speak. End
voice mode, or Escape, stops everything and turns the microphone off.

It is shown where the browser can record (a secure page) and
speech-to-text is on the host. Replies are read only in a voice that
runs on this device; where there is none, it says so and the replies are
shown.

## Why it works this way

- **Everything is written.** What was said and every answer appear in the
  chat as they happen, so nothing is only heard
  ([WCAG 1.2](https://www.w3.org/WAI/WCAG22/Understanding/media-equivalents-audio-description.html),
  [Gemini Live](https://support.google.com/gemini/answer/15274899) gives a
  transcript only at the end).
- **What was heard is said back.** "Sent:" and the words, so a wrong word
  is caught at once without a question to answer
  ([Google conversation design, implicit confirmation](https://developers.google.com/assistant/conversation-design/confirmations)).
- **The microphone is held only while it is about to be used.** It is
  asked for when pressed, not recorded while a reply is read (so the
  assistant does not hear itself), and let go when voice mode waits,
  pauses after 30 seconds with nothing heard, or ends. Then the browser's
  own sign that it is on goes out too, and the words say "The microphone
  is off"
  ([MDN, stopping a track](https://developer.mozilla.org/en-US/docs/Web/API/MediaStreamTrack/stop)).
- **Wait for Speak when replies are not read aloud.** The person, or their
  screen reader, is reading the reply for as long as it takes. Listening
  then would start the 30 second clock and could write down the screen
  reader as their words
  ([Microsoft Voice access FAQ](https://support.microsoft.com/en-us/topic/voice-access-frequently-asked-questions-faqs-c4c2b5ef-5b01-4fbc-af9c-f4b99dec8888)).
- **Headphones, said in plain words.** Without them a screen reader or
  other sound nearby can be taken for the person's words; OpenAI gives
  the same advice for its voice mode
  ([ChatGPT Voice FAQ](https://help.openai.com/en/articles/8400625-voice-mode-faq)).
- **State in words, said once.** Listening, Writing down what you said,
  Sent, Reading the reply, Your turn: said as it changes, and shown. The
  red mark beside Listening is a second sign, not the only one, and
  nothing moves ([WCAG 4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html)).
  The chat's own status says the assistant is working, so this does not
  say it again.
- **Hearing you is shown, not said.** A screen reader speaking while the
  person talks would be recorded into their message.
- **One turn button that stays in place.** Done speaking, Skip the reply
  and Speak are the same button with new words, so pressing it never
  drops the focus; when voice mode pauses or ends the focus goes to the
  start button ([WCAG 2.4.3](https://www.w3.org/WAI/WCAG22/Understanding/focus-order.html)).
- **Done speaking**, for anyone whose pauses are longer than a second, so
  a turn is never cut off by a timer they cannot control
  ([WCAG 2.2.1](https://www.w3.org/WAI/WCAG22/Understanding/timing-adjustable.html);
  [Amazon Nova](https://docs.aws.amazon.com/nova/latest/nova2-userguide/sonic-turn-taking.html)
  waits up to 2 seconds for people who pause).
- **Read replies aloud can be turned off**, and stays off on this device:
  a screen reader already reads the chat's replies, and hearing each twice
  helps nobody. Turning it off during a reply stops the reading.
- **Voices on this device only.** A browser's online voices send the text
  away to be spoken; those are not used.
- **Written down at home**, with the same speech-to-text as recordings and
  dictation; the sound is not kept.

Not done, and why: speaking over the reply to interrupt it (the
microphone would hear the reply itself; Skip the reply does it by a
press, as Gemini Live does when voice interrupting is off); a sound to
mark listening (the words are said and shown, and a sound is one more
thing a screen reader user has to learn); guessing that a screen reader
is running (a page cannot know, and should not); a voice fetched to this
computer where the device has none.
