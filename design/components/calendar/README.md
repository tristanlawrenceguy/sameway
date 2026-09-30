# calendar

Use a calendar to show a month and what is on in it. It is a view: nothing
in it needs a keyboard pattern beyond reading a table and following its
links. To ask someone for a date, use `when-field`.

Give it `month` as `YYYY-MM` and `events` as days with short labels. Mark
`today` and it is called out in words as well as colour. `start` chooses
whether weeks begin on Monday or Sunday.

In a narrow container, such as a side pane, each event is drawn as a marker
rather than a label: at that size a month can honestly answer which days are
busy and not much else. The labels stay in the accessibility tree throughout,
and are drawn again as soon as the calendar has room, so widening it or
moving it into the body of the page is all it takes to read them.

The month grid is worked out on the server, so the component needs no
JavaScript and reads correctly the moment the HTML arrives.

A record that repeats is shown on each day it falls in the month shown,
each with `repeats` in words, "Repeats every Tuesday"; only the day it is
due now carries its actions.

With `type: all` the calendar shows everything with a day together,
from every listed type: tasks due, reminders, entries logged, each
event saying what kind it is. A record's page links to its day on the
first calendar on the canvas, as See that day.

When a month of everything holds more than one kind, a row of links over
it narrows it to one: All (12), Tasks (5), Reminders (4), Entries (3),
each with how many it has in the month, the one shown marked. It is the
[filters](../filters/README.md) component as links, one choice among a
few, applied as it is pressed. The kind is in the page's address, named
after the block (`?c-<block>-type=task`), so two calendars keep their
own, and the months and days either side, and each day number, keep it.
It is offered on the canvas at full size and on the block's own page, and
only narrows: a calendar of one type is never offered another.

A calendar of entries (`type: entry`, or `type: all`) offers in its day
view what it takes to log for that day: each habit it shows, as it stood
that day, with Log and the day filled in. With `where: ["habit=<id>"]`
it is that habit alone. Each entry is a link to its own page, where it
is edited or deleted.

## How a month is read

The caption always names the month, with the caption given or without, so
moving a month on says where it landed. A weekday heading draws "Mon" and
says "Monday" from its own text, the rest of the word hidden after it,
which screen readers read better than an aria-label on a header cell. A
day is said once, with what is on it: "Friday 11 September 2026, 2
events", the number the only part drawn. A day number that opens the day
is pressed over a whole 44px target.

On a narrow screen, where seven columns leave room only for dots, the
month is a list of the days that have something on, each with its words;
the grid and the list are never both shown, so a screen reader meets the
month once. The list links to each event and its day; an event's action
buttons are in the grid only.

In forced colours an event keeps an edge and today its circle, drawn in
the system's own colours.

## Why it works this way

- **It says its month.** The caption always names it, so a person who moves
  a month on hears where they landed
  ([WAI tables tutorial](https://www.w3.org/WAI/tutorials/tables/caption-summary/)).
- **Each day read once.** The date and what is on it are one phrase, and
  the weekday headings say their whole word from their own text, not from
  an aria-label on a header cell, which screen readers read poorly
  ([24 Accessibility on a better calendar](https://www.24a11y.com/2018/a-new-day-making-a-better-calendar/)).
- **A list on a phone.** Seven narrow columns leave room only for dots, so
  a narrow screen shows the days with something on as a list, as a
  calendar app's schedule view does; only one of the two is shown, so it
  is met once ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html)).
- **A day number that opens its day is a 44px target**
  ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).
- **Today in words as well as colour**, and its circle kept in forced
  colours ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **Worked out on the server**, so it reads correctly without a script.
- **A repeat on each day it falls.** A task or reminder that repeats is
  shown on every day it falls in the month shown, as a calendar shows a
  weekly meeting on every week, not only on the day it is next due. Each
  says it repeats in words beside its name, "Repeats every Tuesday", once,
  not with a looping-arrows icon alone, which is not read out
  ([WCAG 1.1.1](https://www.w3.org/WAI/WCAG22/Understanding/non-text-content.html),
  [W3C COGA, use icons that help the user](https://www.w3.org/TR/coga-usable/#use-icons-that-help-the-user-pattern)).
  The days are worked out for the month shown and no further, at most one
  a day, so a repeat that never ends is still a month's worth; they follow
  the repeat's own rules, the 31st on the last day of a shorter month
  ([RFC 5545, 3.8.5.3](https://icalendar.org/iCalendar-RFC-5545/3-8-5-3-recurrence-rule.html)).
- **Only the one due now has its tick.** The later days lead to the same
  record but carry no action: ticking next week's would tick this week's.
- **One kind at a time, by links.** A month of everything is busy; one
  choice among a few kinds is links with counts, applied on press, with
  All first, the one shown aria-current and said in weight and a bar
  ([BBC GEL, filter and sort](https://bbc.github.io/gel/components/filter-and-sort/),
  [NN/g, filters vs facets](https://www.nngroup.com/articles/filters-vs-facets/)).
  The counts are the month's, which is what is on the page.
- **Says what it shows by kind when written.** A calendar of everything
  tells the one who writes it how many of each kind it holds, most first
  ("44 in all: 32 entries, 9 tasks, 3 reminders"), and when one kind is
  more than half, that it crowds out the rest; a model that was told "44
  in all" called them all tasks and reminders. One of a type that shows
  nothing says "nothing yet" and what it waits for.

Not done, and why: a grid you move through with arrow keys (it is a view,
and its links are reached with Tab); the grid and the list both in the
accessibility tree (a screen reader would hear the month twice); the days
a repeat fell on before it was due (those were done, or skipped); skipping
or moving one day of a repeat on its own (it would need exceptions to the
rule, RRULE's EXDATE, and a record for each changed day; a reminder can
skip its next time from the clock, and a task is ticked); a repeating
icon (the words say it); several kinds at once as checkboxes (a form
with Apply for what is one quick choice; All is one press away); kinds
with none in the month (a link to an empty month says nothing a person
can act on).
