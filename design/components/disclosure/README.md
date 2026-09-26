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

`count` is how many are inside, and `of` what they are, said after it to a
screen reader: "Activity, 8 changes". A total kept elsewhere belongs on the
link to the whole of it, not on the summary.

Open or closed is remembered for the page, or with `remember: site` for
every page the disclosure is on, as a canvas's side pane is: closed once,
it stays closed. A reply of the assistant, which swaps the activity log in
anew, keeps it as it was. The body moves in only when a person opens it,
not when a page arrives with it open. On paper, whatever is folded away is
printed too.

The twisty is the one sign, across the design system, that a summary opens
something in place: the folds in a chart and a collection draw the same one
(`design/base/23-twisty.css`). It is drawn with borders, so a screen reader
reads nothing for it.
