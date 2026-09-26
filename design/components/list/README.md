# list

Use a list for several short items of the same kind. Use `ordered` when the
sequence matters. Give it a `label` when no heading directly precedes it, so
assistive technology can name it.

## Why it works this way

- **Its name is a heading everyone sees.** It was an aria-label only, so on
  a trip page sighted people saw two lists with no names, To pack and To
  book, that a screen reader heard; a heading is seen, reached by heading
  and translated ([APG names and descriptions](https://www.w3.org/WAI/ARIA/apg/practices/names-and-descriptions/),
  [Adrian Roselli on aria-label and translation](https://adrianroselli.com/2019/11/aria-label-does-not-translate.html)).
- **Room between items**, so one that wraps reads as one item
  ([GOV.UK lists](https://design-system.service.gov.uk/styles/lists/)).
- **No empty items**: a blank mark and a count that is wrong.
- **Named when removed.** Its Remove says Remove To pack, not Remove 3
  items.
- **Not a to-do list.** Things to tick off are task records in a
  collection, where each has its Done box, saved and undoable
  ([Inclusive Components, a to-do list](https://inclusive-components.design/a-todo-list/)).

Not done, and why: checkboxes in a list (task records with their Done box
already do it, saved and undoable); markers taken away for a cleaner look
(Safari then stops saying it is a list); lists inside lists (short and flat
reads best).
