# move

Moves a record to another choice of a pick-list: a task from To do to
Doing, a project from Active to Done. The choices are a list, Move saves
the one picked, and the page comes back to the record saying where it
went, with an Undo.

A board gives each card one, as one of its actions.

A task moved to Done is ticked done too, and one moved out of Done is
unticked; a task that repeats, moved to Done, is due again on its next day
and back in To do, and says so, as a tick does.

## Why it works this way

- **No dragging.** A select and a button work the same by keyboard,
  switch, touch and voice, with no script
  ([WCAG 2.5.7](https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html),
  [GitHub's testing of a move form](https://github.blog/engineering/user-experience/exploring-the-challenges-in-creating-an-accessible-sortable-list-drag-and-drop/)).
- **Named by what it shows, then the record.** "Move Order compost", so
  "click Move" works and each button is told apart
  ([WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html)).
- **An edit like any other**, logged and undone the same way.
