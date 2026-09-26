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
