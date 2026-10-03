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

## Why it works this way

- **One link.** A screen reader meets one link per card, named by its
  title, instead of a whole card read out as a link's name
  ([Inclusive Components cards](https://inclusive-components.design/cards/),
  [Adrian Roselli on block links](https://adrianroselli.com/2020/02/block-links-cards-clickable-regions-etc.html)).
- **Pressed anywhere.** People expect a card to open wherever they press
  it, so the script follows the title link, but never on a drag to select
  words, a long press or a control inside
  ([NN/g cards](https://www.nngroup.com/articles/cards-component/)).
- **The same cue for a keyboard.** The card rises on keyboard focus as it
  does on hover ([WCAG 2.4.7](https://www.w3.org/WAI/WCAG22/Understanding/focus-visible.html)).
- **Lifted, not only shadowed.** Rising, its surface and edge lighten as
  its shadow deepens: on a dark page a shadow hardly shows, so the lighter
  surface says it, as Material's dark theme does; under forced colours,
  which drop shadows, the edge turns `Highlight`; under reduced motion it
  does not move, the rest still changes
  (design/foundations/elevation.md).
- **Whole titles.** A result is known by its title, so it wraps rather than
  being cut, with the rest in a tooltip no keyboard or finger reaches. The
  title link is a 44px target
  ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).
- **A heading level under its section**, so search results are h3 under
  Results and a collection's cards sit under its own heading
  ([WCAG 1.3.1](https://www.w3.org/WAI/WCAG22/Understanding/info-and-relationships.html)).

Not done, and why: wrapping the whole card in one link (every word becomes
the link's name, and words cannot be selected); a rise on hover for a card
that opens nothing (it promises a press that does nothing).
