# audio

A recording and its words: a voice note, a meeting, a podcast, music.

Without a script the browser's own player plays it. With one, a bar with
words on every control takes its place: Play or Pause, Back 15 seconds,
Forward 30 seconds, Speed, and a seek bar that says "4 minutes 3 seconds
of 12 minutes" when it is moved, not on every tick. Its format, size and
a Download link are always there. It never plays by itself, and starting
one recording pauses any other on the page. If this browser cannot play
it, the page says so in words and the Download link is what is left.

The transcript sits under the player on the same page, a line per stretch
of what was said, with who said it and the sounds that matter. Each line's
time plays from there; without a script it opens the recording at that
moment (`#t=`). While it plays, the line being heard has a border and bold
type. It is never announced and the page is never scrolled to it.

A file's transcript is kept beside the original as a WebVTT file, and its
words are also the file record's text, so search finds them and the
assistant reads them, whoever made the transcript.

## Writing it down

A recording is written down on the computer that hosts the workspace, by
speech-to-text its owner gets once (Get speech-to-text says how much it
downloads, from where, and that recordings never leave). From then on the
page writes each recording down as it opens: its script reads the
recording as the browser plays it and sends the plain sound, so the host
needs no other program. Write it down does the same by hand. With no
script, a WAV can still be written down.

## Why it works this way

- **A transcript, on the page.** It is what anyone who cannot hear the
  recording gets, and the W3C says to put it under the media
  ([WCAG 1.2.1](https://www.w3.org/WAI/WCAG22/Understanding/audio-only-and-video-only-prerecorded.html),
  [W3C on transcripts](https://www.w3.org/WAI/media/av/transcripts/)).
- **Never on its own.** Sound that starts by itself covers a screen reader
  ([WCAG 1.4.2](https://www.w3.org/WAI/WCAG22/Understanding/audio-control.html)).
- **Our own bar, over the browser's.** Browsers' players differ, some
  cannot be reached by keyboard, and Safari's are far below 7:1
  ([Adrian Roselli's review](https://adrianroselli.com/2023/09/browser-video-players-review.html));
  the bar follows the seek slider pattern
  ([APG](https://www.w3.org/WAI/ARIA/apg/patterns/slider/examples/slider-seek/)).
- **Quiet while it plays.** A slider that re-announces its time on every
  tick is the most reported failure of media players
  ([Plyr issue 1794](https://github.com/sampotts/plyr/issues/1794)). So
  while the seek bar has focus it moves only when moved: a value changed
  under it is read out, whatever its words say.
- **One sound at a time.** Two recordings, or one and a screen reader,
  talking over each other are heard as neither
  ([WCAG 1.4.2](https://www.w3.org/WAI/WCAG22/Understanding/audio-control.html)).
- **Says when it cannot play.** A format this browser does not play fails
  on its `<source>`, not on the player
  ([HTML, resource selection](https://html.spec.whatwg.org/multipage/media.html#concept-media-load-algorithm));
  the page puts that into words just above Download, not a Play that does
  nothing ([Adrian Roselli](https://adrianroselli.com/2023/09/browser-video-players-review.html)
  on offering the file).
- **Its title, not over and over.** The figure is named by it, and Play
  and Download carry it for lists of buttons and links. The bar is not a
  named group as well, which said it again on the way in.
- **Back 15, forward 30**, as podcast players do, and speed with the voice
  kept at its pitch.
- **Times as links.** Media fragments work with no script
  ([W3C Media Fragments](https://www.w3.org/TR/media-frags/)).

Not done, and why: a volume control (the device has one, and nothing plays
by itself); a waveform (it moves, and says nothing a time does not);
single-key shortcuts such as Space or K anywhere on the page (they catch
typing and speech commands,
[WCAG 2.1.4](https://www.w3.org/WAI/WCAG22/Understanding/character-key-shortcuts.html));
remembering the speed between recordings (a talk and a song want
different speeds, so each starts at Normal); buttons in place of the
transcript's time links (the links are what works without a script).
