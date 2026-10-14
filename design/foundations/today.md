# Today, and the morning brief

Today (`/today`) is the one page a person opens to know what to do now:
what is late, the tasks due today, the day's events and reminders, then
what came in and waits to be sorted ([sorting.md](sorting.md)). The
morning brief says the same as one notification. Both are read from the records, with no model asked.

## Why it works this way

- **One place, found in one press.** Today is first in the workspace's
  navigation whenever the workspace keeps tasks, events or reminders.
  Things, Todoist, Microsoft To Do (My Day) and Apple Reminders all put
  the day's list at the top of the app; COGA asks that the most important
  tasks of a site be easy to find.
- **A day is the day the person planned.** A whole day is kept as
  midnight UTC, so it is read with `when.Stored`, never as a moment: read
  as one it said "2am", and west of Greenwich a task due today showed as
  late. Moved to tomorrow, a whole day stays a whole day.
- **Overdue says when.** Today's heading is Overdue, the word every
  list, page and the brief use (glance.md). An overdue task says the day
  it was due (Mon 5 Oct), and one with a time says its day and time
  (Yesterday at 3pm), never a bare "3pm" that reads as today's. Overdue
  is plain words, not a red wall: the
  answers sit beside it (Done, Tomorrow, Today) and two or more move as one
  change, the way Todoist's Reschedule works on the Overdue section.
- **A long day starts with three.** With six or more tasks late or due,
  three are shown and the rest folded and counted (the "most important
  tasks" method; COGA, Avoid too much content).
- **What matters starts before it is due.** A task marked important or
  urgent, or of high priority, joins the three when it is due within a
  week or has no day, never the folded rest. Sorting what was in their
  head, a person was told by the assistant to start with the dentist and
  the council tax, and Today went on starting with errands from a
  fortnight ago: two answers to "what first" is one too many.
- **Marked important, said in words.** Microsoft To Do stars a task and
  Todoist flags it red; here the mark is a tag, important or urgent, set
  on the task's page or by asking the assistant. When it puts one among
  the three, the line above them says so once ("Start with these three,
  what is marked important first"), so a task not yet due does not read
  as a mistake. The task type's tags say what the mark does, so an agent
  reading `describe` marks a task the way the assistant does.
- **What the assistant said to start with leads, on any day.** Asked to
  sort a pile, the assistant names the tasks to start with and tags them
  start; Today puts those first, before what is marked important, so the
  page and the chat give one answer to "what now?". On a short day only
  those come first ("The assistant said to start with this") and the rest
  stay as they were, unfolded; on a long one the line says whose order the
  three follow ("what the assistant said first, then what is marked
  important"). A start task leads whatever its day, since the assistant
  said start now. The brief names the same tasks, and the task type's tags
  tell an agent what start does, so one sorting through MCP tags the same
  way.
- **One not due yet is done, not moved.** Among the three, a task that
  matters and is not due yet has Done alone: Tomorrow would bring
  Thursday's dentist forward, or give a day to one that had none, and it
  would still be among the three.
- **The long late can be let go of, in one press.** When two or more tasks
  have been late more than two weeks, Today offers to keep them with no
  date: still tasks, no longer late, all back with one Undo, and the three
  worked out again from what is left. Moving them all to today puts the
  same pile on tomorrow, and going through them one by one is the curating
  a person gets worse at the more overwhelmed they are. What they marked
  as mattering is never among them. Like Things' Someday and Sunsama's
  backlog, they are kept, not thrown away: no longer late, in plain view
  under No date, and the weekly review's Clear the date does the same one
  at a time. OmniFocus keeps a dropped task for the same reason.
- **Each day is a `<time>`** with its value, the page's own date too, so a
  page left open follows itself at midnight and when a time passes
  (`36-days.js`), and a machine reads the date without parsing words.
- **Said once, to everyone.** Each row's title and day are one passage, so
  `look` reads "Renew passport Mon 5 Oct" as a person does, and the presses
  after it name their task once (Done, Renew passport). A folded count
  says what it counts, in the right number: The rest, 3 tasks.
- **The box has its own heading.** "Too much in your head?" is an h2, so it
  is found by heading, and its field is labelled for what goes in it.
- **The day is the person's, in their time zone.** A workspace keeps one
  time zone, a place's name such as Europe/Madrid (`ui.zone`), and
  everything worked out by the day follows it: Today, what is late,
  "tomorrow" said to the assistant or the API, a repeat's next day, the
  morning brief and the weekly review, the day a record was made, a
  calendar and an export. Unset, it is the zone of the computer the
  workspace runs on, which is the person's on their own laptop; but on a
  server in UTC, a Tailscale-hosted box in another country, or a laptop
  that travels, the computer's day is not theirs. Google Calendar,
  Slack and Notion all keep a time zone for the account rather than
  follow the machine, and offer the device's when it differs
  ([Google Calendar](https://support.google.com/calendar/answer/37064?hl=en),
  [Slack](https://slack.com/help/articles/219889247),
  [Notion](https://www.guideflow.com/tutorial/how-to-set-time-zone-automatically-in-notion)).
  The owner sets it on Help; the assistant and an agent with
  `set_setting ui.zone`, or `POST /zone`; describe's `now` says the time
  there with its offset. A change applies to the next thing worked out,
  with no restart.
- **Set without guesswork, never switched silently.** While no zone is
  chosen and the owner's browser is in another (it says its zone by name,
  [MDN, resolvedOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/resolvedOptions)),
  the page asks in plain words near its top: "Times here are this
  computer's, UTC+0: 2 hours behind yours. Use Madrid time for this
  workspace?", with Use Madrid time and Keep this computer's time. Either
  answer chooses (Keep sets `Local`), so it is not asked again; Google
  Calendar likewise asks before it updates a time zone, rather than
  moving the person's days on its own. A viewer who is not the owner is
  only told how far apart the zones are, as before (glance.md).
- **Stored moments stay moments.** A time is kept as an instant and a
  whole day as its date (`when.Stored`), so changing the zone moves no
  record: a 9am meeting kept while in Madrid shows as 8am once the zone
  is London, the same moment, and a task due Friday is still due Friday.
  A repeat at 9am keeps 9am in the zone it is read in, across a change of
  clocks ([RFC 5545, 3.3.5](https://www.rfc-editor.org/rfc/rfc5545#section-3.3.5)).
- **The brief earns its interruption.** It is sent once a day, at the time
  set, and not at all on a day with nothing on. When Today starts with
  three, the brief names those three, so it and the page agree.

## Sources

- W3C, [Making Content Usable for People with Cognitive and Learning Disabilities](https://www.w3.org/TR/coga-usable/): Make it easy to find the most important tasks; Avoid too much content; Limit interruptions.
- NN/g, [Five mistakes in designing mobile push notifications](https://www.nngroup.com/articles/push-notification/): bursts and irrelevant content are why people turn notifications off.
- Todoist, [Introduction to dates and time](https://www.todoist.com/help/articles/introduction-to-dates-and-time-q7VobO), and [Reschedule for overdue tasks](https://www.todoist.com/help/todoist/get-started/plan-your-week-with-the-upcoming-view-OKOg1mR8).
- Things, [Using the Today list](https://culturedcode.com/things/support/articles/4001304/): the day's list reviewed each morning.
- Microsoft, [Plan your day with My Day](https://support.microsoft.com/en-us/office/plan-your-day-with-my-day-4cafa6fe-8891-4ad5-985f-65912146a73e): the day starts fresh, with what is due and overdue offered.
- Leo Babauta, [Purpose your day: most important task](https://zenhabits.net/purpose-your-day-most-important-task/): three things a day.
- Microsoft To Do, [Important](https://windowscentral.com/microsoft-do-windows-10-now-lets-you-mark-tasks-important): one star, its own list; Todoist, [Set a priority](https://www.todoist.com/help/articles/set-a-priority-in-todoist-Wy82Jp): higher priority sits higher in Today.
- Things, [Someday](https://www.culturedcode.com/things/support/articles/2803579); Sunsama, [Task rollover](https://help.sunsama.com/docs/task-rollover-and-recurring-tasks-the-basics): tasks that keep rolling over are archived, kept and reviewable; OmniFocus, [Dropped actions](https://omnigroup.com/blog/omnifocus-dropped-actions): decided against, kept, not deleted.
- Anti-shame overdue handling, as ADHD testers ask for it ([an example](https://github.com/kjswalls/dsul/issues/104)), and a Sameway tester with ADHD: the more overwhelmed, the worse at curating lists.

## Not done, and why

- **A list cleared each night, as My Day does.** Sameway's Today is
  gathered from the records' days, so nothing is lost and nothing needs
  re-adding; the person never curates it.
- **Workload in hours, as Sunsama does.** Tasks here have no estimates, and
  asking for one is more curating.
- **Red for late, or a count badge on the app.** Late is in words with the
  answers beside it; a badge is a standing interruption.
- **A brief on an empty day, to say so.** Today says it to whoever looks;
  a notification that says nothing teaches a person to ignore the brief.
- **Following the device's zone on its own, as Slack and Google
  Calendar can.** The workspace is shared by its person's devices and the
  jobs that run while none is open; a phone abroad changing the day for
  the morning brief at home would be a surprise. A reader in another zone
  is told so on the page instead (glance.md), and the owner chooses.
- **A zone per record, as Todoist's fixed times.** Each time would need
  its zone said beside it; one zone for the workspace covers a person's
  own days ([Todoist](https://todoist.com/help/articles/360001450039)).
- **A calendar export in the zone's own terms (TZID and VTIMEZONE).**
  Times go out as UTC instants, which every calendar reads; a timed
  repeat that crosses a change of clocks may move by an hour there.
- **A list of every zone to pick from.** Some four hundred names; the
  owner types a place's name, and is offered their browser's.
- **A star press on each row, as To Do has.** Today's rows already carry
  Done and Tomorrow; a third press on every row is one more decision on
  the page for the person who is overwhelmed. The tag is set on the
  task's page, by the assistant when sorting, or by asking.
- **A "Not today" for one that matters.** It would need a day's memory
  beside the records; the three are where to start, not a promise, and
  Done or the rest are always there.
- **A separate Someday list.** No date already holds what has no day,
  is reached from the sidebar, and is what the weekly review uses.
- **A start tag that ends on its own.** The assistant takes start off
  what it said before each time it says anew, and a task done leaves
  Today; a day's memory beside the records would be one more thing to keep.
- **Reordering Today by hand.** The three come from what the person marked
  important and from the days; dragging is one more list to keep.
