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
