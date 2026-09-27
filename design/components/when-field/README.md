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

With `repeat` it asks how often instead: "every Tuesday", "every weekday",
"every 2 weeks", "every month on the 1st", "every year on 29 Feb", with "until
1 Mar" if it ends. There is no picker, since a month cannot show a rule; the
words are said back the same way, "Reads as every Tuesday", from `GET
/when?repeat=`. A task or reminder's Repeat field is one of these.

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
- **A repeat in words, not a form of dropdowns.** People say how often
  in a few words, and the tools they know read it from those words:
  Todoist's "every Tuesday", Fantastical's "every Thursday at 7"; the same
  rule as pickers is five choices long. Every common way is read, "daily",
  "fortnightly", "every other week", "each Mon and Fri", and the stored form
  too, so words are not refused for being said another way
  ([Todoist, recurring dates](https://www.todoist.com/help/articles/introduction-to-recurring-dates-YUYVJJAV),
  [W3C COGA, accept different input formats](https://www.w3.org/TR/coga-usable/#accept-different-input-formats-pattern)).
- **Kept as a small RRULE**, the part of iCalendar every calendar exchanges:
  FREQ, INTERVAL, BYDAY, BYMONTHDAY, BYMONTH and UNTIL. Parts it does not
  keep are refused rather than half obeyed
  ([RFC 5545, 3.8.5.3](https://icalendar.org/iCalendar-RFC-5545/3-8-5-3-recurrence-rule.html)).
- **Said back in words that read as the same repeat**: "every month on the
  31st until Mon 1 Mar 2027". "Until 1 Mar" said in September is next March,
  since a repeat does not end before it starts.
- **Errors say what it must be**: how often, a day of the month from the
  1st to the 31st or the last day, a real day (30 Feb is not one), one
  weekday when it is every few weeks, a real day to end on.

Not done, and why: reading the words in the browser as well (two readers
would drift apart; the server's is the one kept); a calendar of our own
(the browser's works with a keyboard and a screen reader already); a repeat
picker of dropdowns, frequency, interval, weekdays, ends (the words are
shorter, and said back); "the second Tuesday" or "the last Friday" of a
month (RRULE can, but few say it; ask for it and it can be added to the one
reader); a count of times instead of an end day (the end day is what people
know).
