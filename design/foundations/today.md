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
- **Late says when.** A late task says the day it was due (Mon 5 Oct), and
  one with a time says its day and time (Yesterday at 3pm), never a bare
  "3pm" that reads as today's. Late is plain words, not a red wall: the
  answers sit beside it (Done, Tomorrow, Today) and two or more move as one
  change, the way Todoist's Reschedule works on the Overdue section.
- **A long day starts with three.** With six or more tasks late or due,
  three are shown and the rest folded and counted (the "most important
  tasks" method; COGA, Avoid too much content).
- **Each day is a `<time>`** with its value, the page's own date too, so a
  page left open follows itself at midnight and when a time passes
  (`36-days.js`), and a machine reads the date without parsing words.
- **Said once, to everyone.** Each row's title and day are one passage, so
  `look` reads "Renew passport Mon 5 Oct" as a person does, and the presses
  after it name their task once (Done, Renew passport). A folded count
  says what it counts, in the right number: The rest, 3 tasks.
- **The box has its own heading.** "Too much in your head?" is an h2, so it
  is found by heading, and its field is labelled for what goes in it.
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
- **Reordering Today by hand.** The three come from what the person marked
  important and from the days; dragging is one more list to keep.
