# clock

The time now, with alarms and timers, and what is coming.

A reminder is a record of the `reminder` type, which every workspace has:
a title, a moment it rings at, whether it is an alarm or a timer, and its
state (set, rang, done). The clock's two short forms make one: Start
timer takes minutes from now, Set alarm takes a time of day and what it
is for. The assistant makes them the same way, with `create_record`.

When a reminder's time comes the server marks it rang and tells every
open page over `/clock/stream`; the clock shows it in an alert region,
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
