# clock

The time now, with alarms and timers, and what is coming.

A reminder is a record of the `reminder` type, which every workspace has:
a title, a moment it rings at, whether it is an alarm or a timer, and its
state (set, rang, done). The clock's two short forms make one: Start
timer takes minutes from now, Set alarm takes a time of day the way a person says it (7:30, 7pm,
tomorrow 6am) and, if wanted, what it is for. Each field has its label in
sight above it, and the alarm time a hint in words; nothing leans on a
placeholder, which is gone the moment a person types. The alarm field
brings up a whole keyboard on a phone, since 7pm and tomorrow are words.
The assistant makes reminders the same way, with `create_record`.

Setting one says what the server understood, where outcomes are said:
"Alarm set. Tea rings tomorrow at 07:00." A time is read the way it was
typed, so 7 is 07:00 and a time already past is tomorrow; saying it back
is how a wrong reading is caught before the alarm fails to ring. Cancel,
Dismiss and 5 more minutes say what they did too ("Tea: 5 more minutes.
Rings again at 14:25."), and each can be undone.

When a reminder's time comes the server marks it rang, whether or not a
page is open: it shows the machine's own notification (a toast, a
banner), and runs the `notify.command` in workspace.yaml if there is
one, with `{title}`, `{text}` and `{url}` in its arguments, which is how
a ring reaches a phone through a push service such as ntfy, or an inbox.
It also tells every open page with a clock over its one connection (`/events?ring=1`, design/base/01-connect.js), kept open while the tab is hidden; the clock shows it in an alert region,
sounds (unless `sound` is false), and raises a browser notification when
the person has allowed them. A ring about something says what, on the
page and in the notification (Water: 3 of 8 glasses so far), and the
notification stays until answered where the browser allows it, and leads
to the thing when pressed. A browser keeps a page silent until the person
has pressed or typed on it once; until then a ring shows and notifies but
makes no sound. It stays until Dismiss (state done) or
5 more minutes (five minutes later, state set again). Without scripts a
rung reminder shows on the next page load, and every control still works.

Under the time, Coming up is one list, soonest first: the reminders
still to ring and whatever is on today from every content type with a
day, a task due, an event, side by side by the hour whatever they are.
The time is a column of its own, the day above it when it is not today
(Tomorrow, Monday, 2 Oct), so the titles line up.
Because reminders are records with a day, the calendar shows them too,
and the calendar's day view shows them by the hour.

Sizes: `glance` is the time and anything ringing, for a header; `full`
adds the forms and the list.

A reminder can be about something. Every record's page has Remind me:
a time, and the reminder is made with `about` set to that page. When
it rings, the notification says the thing and leads to it, and the
thing's page lists the reminders about it. A habit with a `remind` time
nudges the same way: if it is not met by then, the clock rings a
reminder about it once a day, saying where it stands (Water: 3 of 8
glasses so far).

## Why it works this way

- **Says what it set.** A time typed in words can be read wrongly, and
  saying it back is how the person catches that before the alarm fails to
  ring ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **Labels in sight.** A placeholder is gone the moment a person types, so
  each field has its label above it
  ([NN/g on placeholders](https://www.nngroup.com/articles/form-design-placeholders/)).
- **A whole keyboard for the alarm**, not a number pad, since 7pm and
  tomorrow are words
  ([MDN inputmode](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes/inputmode)).
- **One list in time order**, whatever each thing is, with the time in its
  own column, so what comes next is at the top and titles line up.
- **A ring that sounds.** The page wakes its one sound on the first press
  or key, because a browser keeps a page silent until then
  ([Chrome autoplay policy](https://developer.chrome.com/blog/autoplay/)).
- **A ring that waits.** It stays until Dismiss or 5 more minutes, and its
  notification stays until answered where the browser allows, so a person
  who looked away still sees it; each answer can be undone
  ([MDN requireInteraction](https://developer.mozilla.org/en-US/docs/Web/API/Notification/requireInteraction),
  [WCAG 2.2.1](https://www.w3.org/WAI/WCAG22/Understanding/timing-adjustable.html)).

- **A repeat said in words.** A reminder that repeats says so under its
  name, "Repeats every Tuesday", where it rings and in Coming up, not with
  a looping-arrows icon alone, which is not read out and not known to all
  ([W3C COGA, use icons that help the user](https://www.w3.org/TR/coga-usable/#use-icons-that-help-the-user-pattern),
  [WCAG 1.1.1](https://www.w3.org/WAI/WCAG22/Understanding/non-text-content.html)).
- **Dismissed, it is set for its next time**, and the message says when:
  "It repeats every day, so it rings again tomorrow at 07:00." The time
  after is counted on its schedule from when it was due, never from when
  it was dismissed, so an alarm at 07:00 stays at 07:00, five more minutes
  or not; one missed for days rings next at its next time still to come,
  not for each it missed. Across a change of clocks it keeps the time on
  the wall ([RFC 5545, 3.8.5.3](https://icalendar.org/iCalendar-RFC-5545/3-8-5-3-recurrence-rule.html)).
- **Skip this time**, not Cancel, on a reminder that repeats in Coming up:
  it skips the one time and sets the next; Undo puts it back. The whole
  repeat ends by clearing its Repeat on its page, or deleting it.

Not done, and why: a time picker of wheels or dials (typing it as it is
said is faster, and it is said back); a ring that stops by itself (it is
missed by anyone away from the screen); moving a repeating reminder to its
next time the moment it rings (it rings until answered; its next time is
set when it is).
