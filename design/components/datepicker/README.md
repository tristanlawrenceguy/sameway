# datepicker

A native `<input type="date">`: the browser's own calendar, keyboard and
announcements, which a built calendar widget would do worse.

## Use when-field for a date a person knows

Most dates are known: a birthday, a deadline, next Friday. Typing them is
faster than finding them, and a native date input cannot be typed into on a
phone (tapping opens a wheel or a calendar), is hard to reach by voice
control, and on some phone screen readers does not say its errors. So a date
asked for in words is `when-field`, a text field that reads "19 Sep", "next
Friday" or "tomorrow 2pm", with this picker beside it for anyone who would
rather look at a month. It is what a record's page uses for every date.

Use the datepicker alone only when seeing the month is the point: picking a
day by its weekday, or near today, where the calendar is the help.

To show a month and what is on in it, use `calendar` instead.

## Limits

`min` and `max` stop earlier and later days in browsers that honour them,
not in all, so the server checks the value again. When a limit matters, say
it in the hint in words ("From 1 September 2026"), and when a day outside it
is sent, say the rule in the error ("Deadline must be in the future").

Values are ISO (`YYYY-MM-DD`) whatever the browser shows the person, so an
agent sets and reads the same string a person sees. The calendar and its
icon follow the page's light or dark theme, including one chosen outright.

## Why it works this way

- **The browser's own input.** Its calendar, keyboard and announcements are
  ones people already know, and a built widget does them worse
  ([USWDS date picker](https://designsystem.digital.gov/components/date-picker/)).
- **The truth about phones.** A date a person knows is asked for in words,
  with this picker beside it, since a native input cannot be typed into on
  a phone and voice control reaches it poorly
  ([GOV.UK dates](https://design-system.service.gov.uk/patterns/dates/),
  [NN/g date input](https://www.nngroup.com/articles/date-input/)).
- **Dates that do not exist are asked again.** 31 February or 25:00 typed
  beside it is an error to fix, not a day rolled on, and a weekday that
  does not match its date is two facts that disagree
  ([WCAG 3.3.1](https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html)).
- **Dots read as people write them**: 19.9.2026 is a date, 14.30 a time.
- **Limits said in words.** `min` and `max` are not enforced by every
  browser, so the server checks and the hint says the rule
  ([GOV.UK date input](https://design-system.service.gov.uk/components/date-input/)).
- **Follows the chosen theme**, so its calendar and icon match a theme
  picked outright, the icon at full strength
  ([MDN color-scheme](https://developer.mozilla.org/en-US/docs/Web/CSS/color-scheme)).

Not done, and why: a calendar widget of our own (the browser's is better
known and better announced); three boxes for day, month and year (the
words field already takes any date as it is said).
