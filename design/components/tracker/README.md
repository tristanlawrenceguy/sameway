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
habit. `habits` names the ones to show, by name or id, in that order:
`habits: ["Pages per day"]` is a reading tracker with that habit alone
(a name that is no habit's is refused when the block is written, with
the one likely meant). `tags` narrows to habits with one of those tags,
`tags: ["health"]`, of all of them or of those named. `label` names the
tracker: shown over it as its heading and naming its region; left out,
none is shown and the region is called Keeping up. `level` is the
heading's level, 2 to 4, as a collection's is: 2 on a canvas and a
block's own page, straight under the page's title.

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
- **The day to log for is labelled Day**, a word in sight that says what
  goes there; it was "on", which read as a leftover
  ([WCAG 3.3.2](https://www.w3.org/WAI/WCAG22/Understanding/labels-or-instructions.html)).
- **An entry is named by its habit and how much**, Read, 25 minutes, in
  its list, on its page and in the log, never by its id; its day is when
  it happened, Today 12:51, not "was", because it is done.
- **The unit is said once.** The amount's label is the unit drawn beside
  the field, "pages", which says "Amount in pages for Pages per day" to a
  reader: one label, not a hidden one and a drawn one both read, which
  came out as "Amount in pages pages"
  ([WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html)).
- **A tracker shows what it was asked for.** "A reading tracker" is one
  habit, so a tracker picks its habits by name and says its label; before,
  it showed every habit under a name nobody could see.
- **Green means reached**, as in the meter; a limit kept within is in the
  list colour. The run so far is in words, not a warning-coloured arrow.
- **The glance counts what can be met**, leaves readings out, and leads to
  the habits; a habit's standing is named for its day, today or on Mon 21
  Sep.

- **Counted from the first entry.** The days before a habit was kept are
  not misses: the dots, the chart and its numbers start at the first
  entry, so a new habit is not "8 of 30" with twenty-two days it never
  had ([Loop Habit Tracker](https://cdn.jsdelivr.net/gh/isoron/uhabits@dev/README.md)).
- **How often, not only the run.** Under the chart a habit says how often
  it was reached and its best run: "Reached the target of 8 glasses on 21
  of the last 30 days. Best run: 9 days in a row." A broken run shown on
  its own lowers what people do next, more when they blame themselves
  ([Silverman and Barasch, On or Off Track](https://www.colorado.edu/business/faculty-research/2023/04/19/or-track-how-broken-streaks-affect-consumer-decisions));
  one missed day does not undo a habit forming
  ([Lally and others, 2009](https://api.crossref.org/works/10.1002/ejsp.674)).
  The sentence is the chart's text alternative too, visible, with the
  numbers in the table under it
  ([W3C, complex images](https://www.w3.org/WAI/tutorials/images/complex/)).
- **Never twice.** The day after one miss, before today is met, the row
  says "Missed yesterday; today gets back on track", never "0 days"
  ([James Clear, avoid the second mistake](https://jamesclear.com/second-mistake)).
  After two it says nothing of it.
- **A day short says how far it got**: "Tue 10 Mar: 5 of 8 glasses", or
  "nothing logged", rather than "not met".
- **An agent reads the same words.** An entry made with create_record
  says where its habit stands now, as Log does, and get_record on a
  habit gives `standing`: today, the run, how often and the best run.
- **A mistaken entry** is put right with Undo in Log's message, or on the
  entry's own page from the list under the habit.

Not done, and why: rest days and pausing a habit (Apple's Pause Rings,
Habitica's Inn): a habit with no target on some days needs a schedule
first; a routine as a cue ("after lunch") rather than a time: the remind
time is the clock's, and the person can say it in the notes. The entries
under a habit still say their habit beside each, as every list's glance
does; taking it out is the collection's to decide, not one page's.
Saving a streak with tokens or freezes (a game layer; a
day still going already is not a miss); a celebration when a target is met
(motion that distracts; the message says it); a heading when there is no
label (a default heading of Keeping up would repeat the page's own).
