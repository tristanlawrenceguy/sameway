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

Use the datepicker instead when a day must be chosen from a month and never
typed.

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
- **Words first, the picker an option**, hidden without scripts
  ([GOV.UK dates](https://design-system.service.gov.uk/patterns/dates/)).

Not done, and why: reading the words in the browser as well (two readers
would drift apart; the server's is the one kept); a calendar of our own
(the browser's works with a keyboard and a screen reader already).
