# mark

Use a mark for the one press that changes a record's state where a person
sees it: done on a task in a list, pinned on a note on its page. It is a form
with one button that posts `prop-<field>=<value>` to the record's own props
route, so it works without JavaScript, the change is logged and undoable
like any other, and the page comes back where the press happened.

The server puts one on every record that has a yes-or-no field: in a
collection's items, on a record block, on a calendar event at page detail,
and on the record's own page, for the type's first such field. Give
`context` the record's title so each button has its own name.
