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
Dictation stops by itself after five minutes, keeping what was said;
Dictate again carries on from there.

It is shown only where the browser can record: a secure page (https, as
Tailscale gives, or this computer) and a browser with a microphone. The
microphone is asked for when the button is pressed, never before.

## Why it works this way

- **One press to start, one to stop.** Holding a button down while
  speaking, or sliding to lock it, is hard with a switch, a voice, a
  screen reader or a shaking hand; WhatsApp now starts a voice note with
  a single tap too. A press acts when it is let go, so it can be slid off
  ([WCAG 2.5.1](https://www.w3.org/WAI/WCAG22/Understanding/pointer-gestures.html),
  [WCAG 2.5.2](https://www.w3.org/WAI/WCAG22/Understanding/pointer-cancellation.html),
  [WhatsApp's single tap](https://techweez.com/2025/04/30/whatsapp-simplified-voice-recording-feature/)).
- **Nothing sent by itself.** A recording waits in its field and dictated
  words wait in the message, so a person hears or reads what they are
  about to send, and a wrong one is simply replaced
  ([W3C COGA, let users check](https://www.w3.org/TR/coga-usable/)).
- **Words on the button, a quiet timer.** Record changes to Stop recording,
  so its state needs no pressed state beside it and a voice can say what
  it sees ([APG button](https://www.w3.org/WAI/ARIA/apg/patterns/button/),
  [WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html)).
  The time is a timer, which a screen reader does not read on every tick
  ([MDN on the timer role](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/timer_role)).
  The red mark is a second sign, not the only one.
- **Said briefly as it starts.** The status says Recording, or Listening,
  once and in few words: a screen reader speaking over the microphone is
  caught in the recording.
- **A limit that loses nothing.** Dictation stops at five minutes, shown
  beside the time and said as it starts; what was said is written in and
  Dictate carries on, so nothing is lost to the clock
  ([WCAG 2.2.1](https://www.w3.org/WAI/WCAG22/Understanding/timing-adjustable.html)).
  Longer sound would be refused by the computer after the person had
  spoken it all.
- **Asked for when pressed, and every refusal explained.** Not allowed, no
  microphone, and a microphone another program holds are each said in
  words, with how to put it right and what still works (choose a file,
  type the message)
  ([NN/g permission requests](https://www.nngroup.com/articles/permission-requests/),
  [MDN getUserMedia errors](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia#exceptions)).
  A microphone unplugged part way ends the recording and keeps what was
  heard.
- **Unavailable, not silent, while writing.** While words are written down
  the button is marked unavailable and keeps its focus, rather than doing
  nothing when pressed ([APG button](https://www.w3.org/WAI/ARIA/apg/patterns/button/)).
- **Plain words when it fails.** A program's own error is logged on the
  computer, not shown; the person hears that nothing was put in the
  message and can try again or type it.
- **Written down at home.** Dictation uses the same speech-to-text as
  recordings, on the computer that hosts the workspace; nothing is sent
  elsewhere and the sound is not kept. The operating system's own
  dictation (Voice Access, macOS and iOS dictation) types into the message
  too, and nothing here stands in its way.

Not done, and why: holding to talk (see above); stopping when the speaker
goes quiet (it cuts off anyone whose pauses are long; voice mode does it,
with Done speaking); a limit on Record (the file field already says when a
recording is over 64 MB, more than an hour of speech); Escape to stop (the
button keeps focus, and a key that means close elsewhere should not end a
recording); dictating into every field (the operating system does that
already, everywhere); sending as soon as dictation ends (the words are
read first).
