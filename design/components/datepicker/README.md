# datepicker

Use a datepicker to ask someone for a date. It is a native `<input
type="date">` on purpose: the keyboard model, the calendar popup, the
locale-correct display, and the screen reader announcements are the
platform's own, and a built calendar widget would be worse in all four.

The date can always be typed, so nobody is forced through a picker. Values
are ISO (`YYYY-MM-DD`) whatever the browser shows the person, so an agent
sets and reads the same string a person sees.

To show a month and what is on in it, use `calendar` instead.
