# calendar

Use a calendar to show a month and what is on in it. It is a view: nothing
in it is clickable, so nothing needs a keyboard pattern beyond reading a
table. To ask someone for a date, use `datepicker`.

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

With `type: all` the calendar shows everything with a day together,
from every listed type: tasks due, reminders, entries logged, each
event saying what kind it is. A record's page links to its day on the
first calendar on the canvas, as See that day.

A calendar of entries (`type: entry`, or `type: all`) offers in its day
view what it takes to log for that day: each habit it shows, as it stood
that day, with Log and the day filled in. With `where: ["habit=<id>"]`
it is that habit alone. Each entry is a link to its own page, where it
is edited or deleted.
