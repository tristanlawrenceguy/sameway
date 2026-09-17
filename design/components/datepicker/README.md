# datepicker

Use a datepicker to ask someone for a date. It is a native `<input
type="date">` on purpose: the keyboard model, the calendar popup, the
locale-correct display, and the screen reader announcements are the
platform's own, and a built calendar widget would be worse in all four.

The date can always be typed, so nobody is forced through a picker. Values
are ISO (`YYYY-MM-DD`) whatever the browser shows the person, so an agent
sets and reads the same string a person sees.

To show a month and what is on in it, use `calendar` instead.

When a record's date is edited on its page, the words come first: a text
field that reads "19 Sep", "next Friday" or "tomorrow 2pm", with this
picker beside it for anyone who would rather look at a month. People know
the day they mean and type it faster than they find it; the picker is for
the day they do not.
