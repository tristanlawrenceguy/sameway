# disclosure

Use a disclosure for detail that most people, most of the time, do not need
to see: an activity log, raw props, a long aside. It is closed by default and
the summary says what is inside and how many items, so nobody has to open it
to find out.

Native `<details>`, so the state is exposed to every screen reader without
ARIA and the keyboard works everywhere. The enhancement script remembers
whether you opened it, because each action reloads the page.

The content of a closed disclosure is not in the accessibility tree. That is
what makes it quiet, and it is why anything an agent or a screen reader user
must always be able to reach also lives on its own page. The activity log,
for example, is at `/activity` and `/api/activity`.
