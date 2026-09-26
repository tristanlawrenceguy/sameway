# table

Use a table for data with more than one column where rows share the same
columns. The caption is required because it is how the table is announced.
Cells are plain text; for a single column use `list`, and for one item with
several fields use `card`.

The first column names the rows, and is read as each row's header, unless
`rowHeader` is false. Name the columns of numbers in `numbers`, and they line
up on the right. An empty cell says "None" to a screen reader and shows a
dash; a table with no rows says so. Rows are laid under the columns: a short
row is filled out and what is past the last column is left out.

## Why it works this way

- **The first cell heads its row**, so moving across a row hears what it is
  about with each value ([WAI tables, two headers](https://www.w3.org/WAI/tutorials/tables/two-headers/)).
- **Numbers to the right**, so a column of them is compared at a glance
  ([GOV.UK table](https://design-system.service.gov.uk/components/table/)).
- **Headings in sentence case**, not capitals, which read more slowly
  ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **Every cell under a heading**, an empty one said to be empty
  ([WCAG 1.3.1](https://www.w3.org/WAI/WCAG22/Understanding/info-and-relationships.html)).
- **A wide table scrolls in its own box**, which a keyboard can reach, and
  never makes the page scroll sideways; tables written in text do too
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html),
  [Adrian Roselli on responsive tables](https://adrianroselli.com/2020/11/under-engineered-responsive-tables.html)).

Not done, and why: sorting by a column (tables here are short, and a
collection already sorts on the server); rows stacked as cards on a phone
(it strips the table's meaning in some browsers, and a collection's cards do
that job); striped rows (borders and the hover already guide the eye).
