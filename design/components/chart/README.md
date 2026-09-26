# chart

Use a chart when numbers compare or change: how many tasks were done each
week, spend by month, how many of each status. Give it `type` and `by`
(and `period` for a date field, `sum` for a number field, `where` to
narrow) and the server fills the series from records when the page
renders, so the picture is what is true now; or give `series` yourself.
`kind` is `bar` for comparing groups or `line` for change over time.

The picture is drawn on the server as SVG, so nothing needs JavaScript,
and it is never the only copy of the numbers: every value is written on
its bar or point, and the whole series is a real table under the picture,
open at `page` detail and behind a native Numbers disclosure at `full`. A
`glance` is the last value with its trend in words; `brief` adds a small
line. Write `caption` to say what is counted and `description` to say what
the picture shows, in a sentence, for whoever cannot see it.

## Read without the picture

Every chart says what it shows in a sentence under it: the author's
`description`, or, when there is none, one the server writes: where it
starts and ends, its highest and lowest, and how often it reached the
target or kept within the limit. The numbers are a table one press away,
headed with what they are, the target as its last row.

## Drawn for the room

The picture is drawn wide and narrow, and a place shows the one that fits,
so its words are never shrunk below reading size on a phone; the narrow one
leaves the values to the table past seven points. A value below zero is a
bar going down from the zero line. Numbers are written with thousands
separated (12,500), and the unit sits above the axis. In forced colours
the whole picture is drawn in the person's own colours.
