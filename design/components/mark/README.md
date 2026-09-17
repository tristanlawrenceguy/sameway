# mark

Use a mark for a record's one yes-or-no fact where a person sees the
record: done on a task in a list, pinned on a note on its page. It is a
native checkbox inside its label, so a screen reader hears the fact
("Done, checkbox, not checked, Order compost") rather than an action, and
Space changes it. With scripts the change saves itself; without them a Save
button beside it does the same in one more step. Either way it posts
`prop-<field>` to the record's own props route, the change is logged and
undoable like any other, and the page comes back where it was.

The server puts one on every record that has a yes-or-no field, for the
type's first such field: in a collection's items, on a record block, on a
calendar event at page detail, and on the record's own page. Give `context`
the record's title so each checkbox has its own name.

Why a checkbox and not a switch or a button: a checkbox is the control
with the widest support and it states the fact; a switch says the same with
less support and an on-off metaphor that suits settings better; a toggle
button says an action, and its label must not change with its state.
