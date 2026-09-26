# empty

What a place says when there is nothing in it yet. Use it for an empty list,
a search that found nothing, a new canvas or a new conversation, so they all
say it the same way.

Say what would be here and how it gets here in one sentence, and end with the
one thing to do next as the action. Name the action for what it does, such as
"ask the assistant"; never "here", which means nothing read on its own.

Give it a title when the emptiness is the page's news (No notes yet); leave the
title out when the page's own heading already says it. Set `live` only when a
script puts the panel into a page that is already open, as an answer the
person just asked for; a status region that arrives with a new page is not
announced, so a page that loads with nothing found says so in its title
instead, after its name: "Search: plumber, no results".

Say which empty it is, in its words. First use says what goes here and how to
add one (No notes yet. Add one yourself, or ask the assistant). Nothing found
says what was looked for and the way to widen it (No matching tasks. Nothing
is not done and due before today. Try fewer conditions, or see all tasks).
Never give a direction by where something is on the page ("below"); link to
it.

## Why it works this way

- **Says which empty it is.** A list filtered to nothing is not a list with
  nothing yet, and saying "Add one yourself" when there are many misleads;
  each says what would be here and the one thing to do next
  ([NN/g empty states](https://www.nngroup.com/articles/empty-state-interface-design/),
  [Carbon empty states](https://carbondesignsystem.com/patterns/empty-states-pattern/)).
- **Heard when a search finds nothing.** A status region that arrives with
  a new page is never announced, so a page that loads empty says so in its
  window title, the first thing a screen reader says; `live` is for a panel
  a script adds ([WCAG 2.4.2](https://www.w3.org/WAI/WCAG22/Understanding/page-titled.html)).
- **No "below".** A direction by position fails a person who cannot see
  the layout or whose screen reflows it, so the empty panel links to the
  thing instead ([WCAG 1.3.3](https://www.w3.org/WAI/WCAG22/Understanding/sensory-characteristics.html)).
- **The search said as typed**, in plain quotes, so the person sees what
  was looked for ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **Lines stop at a readable length** in a wide panel
  ([WCAG 1.4.8](https://www.w3.org/WAI/WCAG22/Understanding/visual-presentation.html)).

Not done, and why: an illustration in every empty panel (it pushes the
words and the action down, and says nothing a sentence does not); a live
region on a page as it loads (it is not announced).
