# card

Use a card for one item with a title and a short body: a note summary, a
task, a result. Set `href` to link the title to the full item. Choose `level`
so the title sits correctly in the page outline (a card under an h2 section
uses level 3).

## One link, the whole card to press

Only the title is a link: a screen reader hears one link per card, named
by its title, and the words stay selectable. With its script, a press
anywhere on the card follows that link, as people expect a card to; a drag
to select words, a long press or a press on a control inside the card does
not. Ctrl or Cmd opens it in a new tab. The card rises on hover only then,
and on keyboard focus always, so a keyboard sees what a pointer sees.

## Several cards

Put several cards in a list, one item each, so a screen reader says how many
there are. Cards suit mixed things; for many items of one kind that a
person compares or hunts through, a list or a collection reads faster.

Keep the title first in the markup; to show the meta above it, reorder with
CSS, never by moving the markup.
