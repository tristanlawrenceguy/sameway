# Glance: what a record says beside its title, and how a day is said

A record is met far more often in a list than on its own page, so the few
words beside its title decide whether a person opens it. Those words are
worked out once, from the schema (`internal/server/glance.go`), and every
surface says the same facts its own way: chips under the title on its page,
short words at the right of its row, plain words in a list on the canvas,
and `glance` over the API and in the assistant's `get_record`. Days and
times are said by one package (`internal/when`), everywhere.

## What a record says

In this order, always, so the same fact sits in the same place on every
row and a person can compare rows by looking down a column
([NN/g, list entries](https://www.nngroup.com/articles/list-entries/)):

1. **Its tick**, when ticked (Done). A row or a page with a box does not
   say it again.
2. **Its state, when it says something**: a stage beyond the tick (Doing),
   a status or state away from where every record rests (Published, not
   Draft; Ringing, not Waiting), or one that waits on someone (Pending). A
   kind, a method, a cadence is the record's own page's to say. Few states,
   short words, sentence case
   ([GOV.UK, complete multiple tasks](https://design-system.service.gov.uk/patterns/complete-multiple-tasks/)).
3. **A setting that is on**, by its label (Pinned, On the canvas, Archived).
4. **Its first day**, named by its field: Due Fri 9 Oct, Starts tomorrow
   at 2pm, Goal by Mon 4 Jan 2027. A field called at or on is not named
   (Today at 7am, not At today at 7am).
5. **What it belongs to or who it is for**, unless that is its title: the
   first reference with a value. A file it points at (a meeting's
   recording) is an attachment, not what it belongs to, and is not said.

At most these five, usually two or three: enough that a person need not
open each record to tell them apart, few enough to scan
([NN/g, list entries](https://www.nngroup.com/articles/list-entries/),
[NN/g, information scent](https://www.nngroup.com/articles/information-scent/)).
A row of a type with no day says when it last changed instead; a record's
own page says when it was made.

Which field is a type's day is the schema's one answer (`DayField`):
`starts` when it has one, else its first shown datetime field. A list
groups its rows by it, a calendar puts records on it, an export's
calendar file writes it, and what else is on a record's day is found by
it, so they never disagree. The glance says the first day a record has a
value in, named by its field, so an interaction with only a follow up
still says it, though its row sits under No date.

| Type | At a glance |
|---|---|
| task | Done, Doing, Due Fri 9 Oct at 2pm (Overdue, due yesterday), For Ana Silva or its project |
| note | Published, Pinned; a row says when it changed |
| habit | Archived, Goal by Mon 4 Jan 2027 |
| entry | Today at 8am (when it happened, never late), its habit |
| event | Starts tomorrow at 2pm (past: neutral, never late) |
| reminder | Ringing or Done, Today at 7am |
| person | when it changed (a person's organisation is a string the schema does not mark) |
| project | Done |
| file | Being read, Could not be read |
| action | On the canvas |

## Late, and today

- **Late is said in words**: "Overdue, due yesterday", in amber, never
  amber alone ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
  A row in its type's list keeps the short day, under the list's Overdue
  heading, which says it once for the group.
- **Only what can be done is late.** A record with a done tick that is not
  ticked and whose day has passed. A meeting that happened, an entry
  logged, a reminder that rang are past, not late, and are not amber.
- **A day is late once the reader's today is after it; a moment, once its
  time has gone.** Today is amber with the word Today
  ([Todoist, dates](https://www.todoist.com/help/articles/introduction-to-dates-and-time-q7VobO)).

## How a day is said

- **A day planned for** (due, starts, rings) is Today, Tomorrow, Yesterday,
  else its weekday and date, "Fri 9 Oct", with the year only when it is not
  this one. People plan against these, so they lean absolute; "In 4 days"
  makes a person count and is wrong on a page left open
  ([UX Movement, absolute vs relative timestamps](https://uxmovement.com/content/absolute-vs-relative-timestamps-when-to-use-which/),
  [Cloudscape, timestamps](https://cloudscape.design/patterns/general/timestamps/)).
- **When something happened** (made, changed, deleted) is said by how long
  ago while that is short, "Today at 2pm", "Yesterday at 9:30am", "3 days
  ago", then its date, "2 Oct" ([Atlassian, date and time](https://atlassian.design/foundations/content/date-time),
  [Primer, relative time](https://primer.style/components/relative-time)).
  Nothing says minutes ago: the words change at most once a day.
- **Day, month, year**, the month a word, no commas, no ordinals: "Sat 19
  Sep 2026" in full, which the day field reads back as the same value
  ([GOV.UK style guide, dates](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/style-guides/a-to-z-style-guide/)).
- **A day alone has no time** and is the same day everywhere (it is kept as
  that date); an all-day event shows no time.
- **Headings and short dates are said by `internal/when` too**: a month
  "September 2026" (`Month`), a day over what happened on it "Today,
  Wednesday 7 October" (`DayHeading`), a date beside a name "2 Oct", "2 Oct
  2025" (`Date`), a message's time alone today and "5 Oct at 2pm" before
  (`Sent`). A record taken out as text (Markdown, a published record over
  MCP) says its days in full, "Friday 9 October 2026", by `export.Text`.

## How a time is said

- **The person's clock.** On the 12-hour clock "2pm", "5:30pm", "midday",
  "midnight", never "12pm" or "2:00pm"; on the 24-hour clock "14:00",
  "09:30" ([GOV.UK style guide, times](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/style-guides/a-to-z-style-guide/)).
  Which is the person's: `ui.clock` (12 or 24, on the Help page), else the
  workspace's language (12 for English, as GOV.UK writes; 24 for German,
  French and most others, as CLDR's hour cycles say)
  ([MDN, Intl.DateTimeFormat hourCycle](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/DateTimeFormat),
  [Atlassian](https://atlassian.design/foundations/content/date-time)).
  The clock's face keeps its minutes (2:00pm, 2:05pm), since it ticks.
- **The workspace's zone.** Every time is the computer's the workspace runs
  on, which for a personal workspace is the person's. When a browser reads
  it from another zone, the page says so once under its heading: "Times
  here are this workspace's, UTC+2: 1 hour ahead of yours"
  ([GOV.UK, "UK time"](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/style-guides/a-to-z-style-guide/),
  [Google Calendar, time zones](https://support.google.com/calendar/answer/37064?hl=en)).

## In the page

- **A `<time datetime>` holds every day and moment** a glance, a field or a
  made-line says: the date alone for a day, the moment with its offset
  otherwise, for a machine. A screen reader reads the words, which say it on
  their own; screen readers mostly ignore `datetime` and announce times
  unreliably when their text is overridden, so the words are never
  replaced ([MDN, time](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/time),
  [Adrian Roselli](https://adrianroselli.com/2023/04/dont-override-screen-reader-pronunciation.html)).
  When the words leave the date out (Today, 3 days ago) its `title` holds it
  in full, for a pointer.
- **Days stay true on a page left open.** At the workspace's midnight, and
  when a time the page shows passes (a task falls due), the page follows
  itself as it does after a change (`30-days.js`, `17-refresh.js`),
  keeping scroll, focus and typing; a hidden tab waits until it is seen
  ([Primer](https://primer.style/components/relative-time),
  [UX Movement](https://uxmovement.com/content/absolute-vs-relative-timestamps-when-to-use-which/)).

## Said once

The lede says the glance; the fields under it leave out what it said, the
title, the box's field, a state not worth saying, so nothing is read twice
([GOV.UK summary list](https://design-system.service.gov.uk/components/summary-list/)).
A reference keeps its field, a link there being a way the chip is not.
Agents read the same words: `glance` on a record over the API and from
`get_record`, and `look`'s text, which reads the page.

## Not done, and why

- **Each reader's own zone on the server.** Rendering in the browser's zone
  needs every surface's now to come from the request; a personal workspace
  is read mostly in its owner's zone, so the page says the zone when it
  differs instead.
- **The record's own zone** (an imported meeting in New York time). Times
  are stored as moments; the zone they were written in is not kept.
- **Times written in the browser's locale by script** (Intl). Pages are
  server-rendered and agents read them without scripts; the words would
  differ between the two.
- **"In 4 days", "in 3 hours".** Counting is the reader's work and the
  words go stale by the minute.
- **A tooltip on every date.** Only where the words leave the date out; a
  title on every date is read again by some screen readers.
- **"11:59pm" for a deadline at the end of a day.** A day alone is the whole
  day; a timed deadline at midnight is said "midnight", as written.
- **Glance hints in the schema** (marking a person's organisation as worth a
  glance). The rules read the schema's types and names; a hint would be a
  new schema word for every agent to learn.
