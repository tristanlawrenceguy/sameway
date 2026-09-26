# mark

Use a mark for a record's one yes-or-no fact where a person sees the
record: done on a task in a list, pinned on a note on its page. It is a
native checkbox inside its label, so a screen reader hears the fact
("Done, checkbox, not checked, Order compost") rather than an action, and
Space changes it. With scripts the change saves itself; without them a Save
button beside it does the same in one more step. Either way it posts
`prop-<field>` to the record's own props route, the change is logged and
undoable like any other. With scripts it saves where it is, focus staying on the box; without, the page comes back where it was.

The server puts one on every record that has a yes-or-no field, for the
type's first such field: in a collection's items, on a record block, on a
calendar event at page detail, and on the record's own page. Give `context`
the record's title so each checkbox has its own name.

Why a checkbox and not a switch or a button: a checkbox is the control
with the widest support and it states the fact; a switch says the same with
less support and an on-off metaphor that suits settings better; a toggle
button says an action, and its label must not change with its state.

## Why it works this way

- **Saved where it is.** Ticking keeps focus on the box and saves in the
  background, so a list is ticked down without the page reloading to its
  top ([WCAG 3.2.2](https://www.w3.org/WAI/WCAG22/Understanding/on-input.html)).
- **Said, and said specifically.** "Sow beans is done.", through a status
  region that is on the page from the start, since one put in with its words
  is often not read out; its Undo names the thing
  ([Inclusive Components, a to-do list](https://inclusive-components.design/a-todo-list/),
  [WCAG 4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html)).
- **The message stays** until closed or left, not replaced by the page
  catching up.
- **A refused save is not shown as saved.** The box goes back and the reason
  is said as an alert.
- **Nothing moves under the person.** A ticked row stays where it is, struck
  through; the list is brought up to date when focus leaves it
  ([GOV.UK task list](https://design-system.service.gov.uk/components/task-list/)).
- **Ticked twice quickly, saved as it ends up**, not in whichever order the
  two saves arrive; the box is never disabled while it saves
  ([Adrian Roselli on disabled controls](https://adrianroselli.com/2024/02/dont-disable-form-controls.html)).

Not done, and why: a switch (a switch is for a setting; a checkbox states a
fact about the thing); a button that toggles (it names an action, and loses
the form that works without scripts).
