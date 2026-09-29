# move

Moves a record to another choice of a pick-list: a task from To do to
Doing, a project from Active to Done. The choices are a list, Move saves
the one picked, and the page comes back to the same Move button, now in
the record's new column, saying where it went, with an Undo.

A board gives each card one, as one of its actions. Give it an `id` for
its Move button: the form sends it as the place to come back to.

Use it when a record sits in a group by a pick-list and a person wants it
in another. Not for a yes-or-no (a mark) or several fields at once (the
record's editor).

## What a screen reader hears

Tab from the card's title reaches the choices, "Status, combo box,
Active", then "Move House, button". After Move the page comes back with
focus on that button, in the Done column, and its description says
"House moved from Active to Done." once. The same message, with Undo,
sits at the top of the page for anyone who wants it.

A task moved to Done is ticked done too, and one moved out of Done is
unticked; a task that repeats, moved to Done, is due again on its next day
and back in To do, and says so, as a tick does.

## Why it works this way

- **No dragging.** A select and a button work the same by keyboard,
  switch, touch and voice, with no script. Voice users found GitHub's move
  form the way they preferred
  ([WCAG 2.5.7](https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html),
  [GitHub's testing of a move form](https://github.blog/engineering/user-experience/exploring-the-challenges-in-creating-an-accessible-sortable-list-drag-and-drop/)).
- **Named by what it shows, then the record.** "Move House", so
  "click Move" works and each button is told apart
  ([WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html)).
- **Pick, then press.** Choosing does not send by itself, so moving
  through the choices by arrow keys moves nothing
  ([WCAG 3.2.2](https://www.w3.org/WAI/WCAG22/Understanding/on-input.html)).
- **Focus returns to the Move button pressed**, in the record's new
  column, so the next card to move is near, not at the top of the page
  ([Atlassian drag and drop accessibility](https://atlassian.design/components/pragmatic-drag-and-drop/accessibility-guidelines)).
  Without scripts the browser still scrolls there.
- **Where it went, said once, from and to.** The button's description is
  the message while focus is on it, then goes. A message already on the
  page when it loads is not read out, so this is how it is heard
  ([Atlassian](https://atlassian.design/components/pragmatic-drag-and-drop/accessibility-guidelines),
  [Scott O'Hara, are we live](https://www.scottohara.me/blog/2022/02/05/are-we-live.html)).
- **An edit like any other**, logged and undone the same way.
- **A move to where it already is saves nothing** and says "Status is
  already Done. Nothing changed.", with no Undo for a change never made.

## Not done, and why

- **Dragging as well.** It would be a second way for pointers only, and
  needs a script, keyboard mode and live announcements of its own to be
  fair to everyone. The form already is the way that works for all.
- **A button for each column** ("Move to Done"). Neat with three columns,
  but a card grows a row of buttons with every choice added, each said in
  turn. A list stays one stop for any number of choices.
- **An actions menu**, as Atlassian and GitHub offer. It needs a script to
  open and a menu pattern to learn; a select is native everywhere.
- **Moving up or down within a column.** A board is grouped by a
  pick-list, with no order of its own to keep.
- **The record's title in the select's label.** The select's label is the
  field's, "Status"; the card's title link comes just before it, and the
  button carries the title. Adding it needs the select component to take
  hidden words.
- **Moving without the page coming back**, as a mark does. The card has
  to change column, which is the whole page's work; coming back to the
  button gives the same place, heard the same way.
