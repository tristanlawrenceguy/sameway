# tracker

What is being kept up, at a glance, with a press to log.

Two provided types carry it. A `habit` is something to keep up: a name,
a cadence (day or week), a target for each period (1 for once, 8 for
eight glasses, 5 for five kilometres), a unit when it is measured, and
an optional longer goal with a date. An `entry` is one thing logged
against it: when, and how much. The Log press on the tracker makes an
entry; so does the assistant with `create_record`; so does an import
of a CSV of dates and amounts.

The server reads the entries and fills the tracker in: this period
against the target, met or not; the streak (counting back from today
when today is met, from yesterday when it is not, since a day still
going is not a day missed); the best run there has been; the last seven
days or four weeks as dots; and how far a goal has come. The habit's
own page has the same numbers with a chart of the last thirty days, the
target drawn across it.

Sizes: `glance` is how many are met this period; `full` is every
habit. `tags` narrows to habits with one of those tags.
