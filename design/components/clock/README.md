# clock

The time now, with alarms and timers, and what is coming.

A reminder is a record of the `reminder` type, which every workspace has:
a title, a moment it rings at, whether it is an alarm or a timer, and its
state (set, rang, done). The clock's two short forms make one: Start
timer takes minutes from now, Set alarm takes a time of day the way a person says it (7:30, 7pm,
tomorrow 6am) and what it is for. The assistant makes them the same way, with `create_record`.

When a reminder's time comes the server marks it rang, whether or not a
page is open: it shows the machine's own notification (a toast, a
banner), and runs the `notify.command` in workspace.yaml if there is
one, with `{title}`, `{text}` and `{url}` in its arguments, which is how
a ring reaches a phone through a push service such as ntfy, or an inbox.
It also tells every open page over `/clock/stream`; the clock shows it in an alert region,
sounds (unless `sound` is false), and raises a browser notification when
the person has allowed them. It stays until Dismiss (state done) or
5 more minutes (five minutes later, state set again). Without scripts a
rung reminder shows on the next page load, and every control still works.

Under the time, Coming up lists the reminders still to ring and whatever
is on today from every content type with a day: a task due, an event.
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
