# Weekly review: looking back without blame, then planning

A list kept for weeks fills with what is overdue, and nobody looks back.
The weekly review (`/review`, `internal/server/review.go`) is one page
for it, and the day it says it is ready is chosen on Help
(`review_day.go`).

## Why it works this way

- **What was done comes first.** The week is said in one sentence under
  the title, done before overdue: "The seven days to Saturday 10 October:
  3 tasks done, 2 overdue and 4 due in the week ahead." Then the lists in
  that order. A retrospective opens with what went well, so the rest is
  read as a plan, not a verdict
  ([Kerth's prime directive, Parabol](https://www.parabol.co/blog/retrospective-prime-directive/));
  Sunsama's review leads with the work done
  ([Sunsama, Weekly Review 2.0](https://roadmap.sunsama.com/changelog/weekly-review-20)).
- **Counts are said once, in words, and none is left out.** The sentence
  is the only place the counts are; a section with nothing says so in
  plain words, and one with nothing to act on (Overdue, Habits) is not
  shown at all ([W3C COGA, avoid too much content](https://www.w3.org/WAI/WCAG2/supplemental/patterns/o5p03-manageable-quantity/)).
  Numbers are numerals ([GOV.UK style guide](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/style-guides/a-to-z-style-guide/)).
  There is no chart: a week is a handful of numbers, which a sentence
  says better.
- **Overdue is the app's one word for it**, as in every list
  ([glance.md](glance.md)), not "slipped". Each overdue task has three
  ways on, one press each with its Undo: Move to next week, Clear the
  date (keep it for later), Done. Clearing the date is GTD's
  renegotiating a commitment: something put off week after week is
  better kept with no date than carried forward again
  ([David Allen, renegotiate your commitments](https://gettingthingsdone.com/2020/05/renegotiate-your-commitments/)).
  A wall of red overdue tasks is where people give up on a list
  ([anti-shame reschedule UX](https://github.com/kjswalls/dsul/issues/104));
  forgiving a delay makes the next one less likely
  ([Wohl, Pychyl and Bennett, 2010](https://www.academia.edu/28728632/I_forgive_myself_now_I_can_study_How_self_forgiveness_for_procrastinating_can_reduce_future_procrastination)).
- **The week ahead includes the calendar.** GTD's review looks at the
  calendar ahead as well as the lists
  ([Todoist, the weekly review](https://www.todoist.com/productivity-methods/weekly-review));
  the next seven days of events are under On the calendar, across a
  month's end.
- **Plan next week asks the assistant to run a retrospective**: what went
  well, what got in the way, then what matters next. A person who wants
  to write the week down does it there or in a note; the page adds no
  journal of its own.
- **The notification says what was done, never what is late.** "3 tasks
  done this week. Look back, and plan the next one." A lock screen that
  counts what slipped is a scolding. It comes once, on the day the owner
  picks, at 6pm on their clock, and Help says what it will say and when
  before it is set, with No reminder as a choice
  ([NN/g, push notification mistakes](https://www.nngroup.com/articles/push-notification/)).
  Todoist suggests Friday afternoon, Sunday evening or Monday morning;
  the day is the person's.
- **An agent does the same.** `look` on /review reads the sentence and the
  lists under their headings; the three presses are `update_record`, the
  day is `set_setting review.on`.

## Not done, and why

- **No completion rate or score.** A percentage of tasks done turns a
  review into a mark; the count of what was done is enough.
- **No "most pushed" list.** Sunsama shows the tasks moved most often;
  Sameway keeps no count of moves, and Clear the date already answers
  the task that keeps moving.
- **Done this week is a task ticked and last changed in the seven days.**
  A repeating task ticked this week is due again, so it is not in Done;
  its tick is in the log. Reading the log for it is more than a weekly
  page needs.
- **No choice of hour.** Six in the evening suits the days people pick
  (Friday, Sunday); a second setting would be read by few.
