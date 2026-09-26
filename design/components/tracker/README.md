# tracker

What is being kept up, at a glance, with a press to log.

Two provided types carry it. A `habit` is something to keep up: a name,
a cadence (day, week, month or year), a target for each period (1 for
once, 8 for eight glasses, 5 for five kilometres), a unit when it is
measured, and an optional longer goal with a date.

Its `aim` says which way the target points. `reach` (the default) is at
least the target. `limit` is at most: an allowance of hours a month, a
weekly budget, screen time a day. The row says what is left, or how far
over, and a period counts as met while it stays within; the bar turns
the warning colour past it. `record` has no target at all, no bar and no
streak, only the numbers, for a measure such as weight.

Its `combine` says how a period's entries make its amount: `sum` (the
default) adds them up; `latest` is the last one logged, for a reading;
`average` is their mean, for hours slept a night over a week. A period
with nothing logged has no reading rather than a reading of nought. An `entry` is one thing logged
against it: when, and how much. The Log press on the tracker makes an
entry, and on a habit's own page it also offers a day, for something
done before today (yesterday, 22 Sep, 2026-09-22; empty is now); so does the assistant with `create_record`; so does an import
of a CSV of dates and amounts.

The server reads the entries and fills the tracker in: this period
against the target, met or not; the streak (counting back from today
when today is met, from yesterday when it is not, since a day still
going is not a day missed); the best run there has been; the last seven
days, four weeks, six months or three years as dots; and how far a goal has come. The habit's
own page has the same numbers with a chart of the last thirty days,
twelve weeks, twelve months or five years, the target (or the limit)
drawn across it; a line rather than bars when the amount is a latest or
an average.

Sizes: `glance` is how many are met this period; `full` is every
habit. `tags` narrows to habits with one of those tags.

A habit aimed to reach with a `remind` time (20:00, 8pm) nudges through the clock: if
it is not met by then, a reminder about it rings once a day, saying
where it stands, and leads to the habit. An entry is titled by its
habit and amount (Water: 8 glasses), so it reads on the calendar and in
a list.

## Why it works this way

- **Log says what it did and where it stands**: "Water: 1 glass logged.
  Now 6 of 8 glasses.", with Undo there; before, the page just reloaded
  ([NN/g, visibility of system status](https://www.nngroup.com/articles/visibility-system-status/),
  [NN/g, user control and freedom](https://www.nngroup.com/articles/user-control-and-freedom/)).
- **Logged where it is.** With scripts, Log saves without leaving the page,
  focus staying on it for the next glass; the list catches up when focus
  leaves it ([WCAG 3.2.2](https://www.w3.org/WAI/WCAG22/Understanding/on-input.html)).
- **A day still going is not a day missed**: its dot is a dashed ring and
  says how far it has got; days are named as a person says them, Wed 16 Sep,
  not 2026-09-16 ([GOV.UK dates](https://design-system.service.gov.uk/patterns/dates/)).
- **Green means reached**, as in the meter; a limit kept within is in the
  list colour. The run so far is in words, not a warning-coloured arrow.
- **The glance counts what can be met**, leaves readings out, and leads to
  the habits; a habit's standing is named for its day, today or on Mon 21
  Sep.

Not done, and why: saving a streak with tokens or freezes (a game layer; a
day still going already is not a miss); a celebration when a target is met
(motion that distracts; the message says it).
