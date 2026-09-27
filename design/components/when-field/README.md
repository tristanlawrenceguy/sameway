# when-field

Asks when something is, the way a person says it: 19 Sep, next Friday,
tomorrow 2pm. People know the day they mean and type it faster than they find
it in a month; the platform's own picker stands beside the words for anyone
who would rather look, and picking a day writes it into the words, keeping any
time already typed.

The words are what is sent and the server reads them. The picker sends nothing
and is hidden until scripts run, since without them it could not write into
the words. Pass `day` as YYYY-MM-DD so the picker opens on the day already
there. The hint is always shown: it is the example of what can be typed.

It is the one component for asking for a date. To show a month and what is
on in it, use `calendar`.

## Why it works this way

- **Said back as it was read.** Once the words are changed, the field says
  how the server read them, "Reads as Fri 2 Oct 2026, 14:00", or what they
  must be; after saving, the message says it again. "Next Friday" and "3
  Jan" can each mean two days, and saying which is how a wrong one is
  caught ([Adrian Roselli, maybe you don't need a date picker](https://adrianroselli.com/2019/07/maybe-you-dont-need-a-date-picker.html),
  [W3C COGA, feedback](https://www.w3.org/TR/coga-usable/)).
- **Errors say what it must be**: a day at all, a real day (31 Feb is not
  one), a real time, or one day where the weekday and the date disagree
  ([GOV.UK date input](https://design-system.service.gov.uk/components/date-input/)).
- **The picker says what it is** in words beside it, since some browsers
  draw it with no calendar icon, and it does not jump open on focus
  ([Hassell Inclusion on collecting dates](https://www.hassellinclusion.com/blog/collecting-dates-accessible/)).
- **Picking a day keeps the time typed**, however it was written: 2pm,
  14:00, 14.30, noon, midday.
- **Words first, the picker an option**, hidden without scripts, since a
  native date input cannot be typed into on a phone, voice control reaches
  it poorly, and some phone screen readers do not say its errors
  ([GOV.UK dates](https://design-system.service.gov.uk/patterns/dates/),
  [NN/g date input](https://www.nngroup.com/articles/date-input/)).
- **The browser's own picker**, not a built one: its calendar, keys and
  announcements are ones people already know
  ([USWDS date picker](https://designsystem.digital.gov/components/date-picker/)).
  It follows a theme chosen outright, its icon at full strength
  ([MDN color-scheme](https://developer.mozilla.org/en-US/docs/Web/CSS/color-scheme)).

Not done, and why: reading the words in the browser as well (two readers
would drift apart; the server's is the one kept); a calendar of our own
(the browser's works with a keyboard and a screen reader already); a
separate date component that is only the picker (it was retired: it let a
date be asked for with no words to type and nothing said back, and a block
saved with it now shows as this field).
