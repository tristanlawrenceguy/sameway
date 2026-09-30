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

## Why it works this way

- **Says what it shows.** A chart with no description still gets a
  sentence the server writes, since a picture alone tells a person who
  cannot see it nothing
  ([WAI complex images](https://www.w3.org/WAI/tutorials/images/complex/),
  [Chartability](https://chartability.fizz.studio/)).
- **The numbers as a table**, headed with what they are, so every value
  can be read, copied and compared
  ([WCAG 1.1.1](https://www.w3.org/WAI/WCAG22/Understanding/non-text-content.html)).
- **Readable on a phone.** A narrow picture is drawn for narrow places, so
  labels are never shrunk to a size nobody can read
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html)).
- **Honest below zero.** A negative value is a bar down from the zero line,
  not an empty bar that reads as nothing
  ([Analysis Function charts guidance](https://analysisfunction.civilservice.gov.uk/policy-store/data-visualisation-charts/)).
- **Values on the bars**, so colour and height are never the only way to
  read them ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **Drawn in the person's colours** in forced colours.
- **A date needs a period.** A chart by a date with no `period` is refused
  when it is written, with the days its records fall on, so the writer
  picks day, week or month; left to a default, one monthly bar of 142 was
  told to a person as their daily water. Its line then says which: "30
  days, 2026-09-01 to 2026-09-30". A chart stored before this still groups
  by month. Conditions that ask one field for two values are refused too,
  as a list's are, and a chart with nothing to draw says "nothing yet"
  and what it waits for.

Not done, and why: a chart drawn by a script in the browser (the server's
SVG reads the moment the page arrives, with no script); the narrow picture
labelling every point past seven (the words would not fit; the table has
them).
