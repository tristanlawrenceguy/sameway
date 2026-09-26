# record

Use a record to put a content record on the canvas as the record it is. A
note shown this way and the note on `/t/note` are one thing: edit the title
in either place and the other changes, because both post to the same
`/t/<type>/<id>/props`. A card with the note's words copied into it is not
this; it drifts from the note the moment either one changes.

Give it `type` and `record`, nothing else. The server fills in the title,
the text and the other fields from the record when it renders the page, and
says in words when the record no longer exists. Choose `detail`: `brief` is
the title alone, for a pane or a list of many; `full` is the title, the text
and the fields; `page` adds the link to the record's own page, and is what a
block gets when it is expanded. Set `level` so the title sits correctly in
the page outline.

## Why it works this way

- **Edit on the canvas is the record's whole editor.** Every field of the
  record, the empty ones behind Add, as on its own page; before, only the
  title and the text could be changed there, and the facts vanished while
  editing ([GOV.UK summary list](https://design-system.service.gov.uk/components/summary-list/)).
- **Named by its title.** Its Edit, Remove and form say "Edit task Order
  compost", not the record's id ([WCAG 2.4.6](https://www.w3.org/WAI/WCAG22/Understanding/headings-and-labels.html)).
- **Its kind in words**, Task or Note, with the list's dot beside it: the
  dot alone said it only in colour
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **A gone record gives the way on**: it may have been deleted, and the rest
  of its list is a link away ([GOV.UK page not found](https://design-system.service.gov.uk/patterns/page-not-found-pages/)).
- **Escape does not lose what was typed.** The block says it was being
  edited and offers it again; only Cancel lets it go
  ([W3C COGA, avoid data loss](https://www.w3.org/TR/coga-usable/)).
- **Closing the saved message returns focus to the Edit** it came from, not
  the top of the page.

Not done, and why: a Change link on every fact (many small targets on a
canvas of blocks; one Edit reaches them all); saving each field as it is
left (the full form, its draft and its Undo already keep work safe).
