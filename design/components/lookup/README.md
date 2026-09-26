# lookup

Choose one record from many by typing part of its name. The records that match
are offered as suggestions as the person types, drawn by the browser's own
datalist, and the one chosen is sent by its id. The box always shows the
record's name, never its id.

Use it for a link to another record when there are too many to list, such as
a person out of thousands; the inline editor switches to it on its own past
500. For a short fixed set use a select.

It needs scripts to suggest (it asks `/api/search`); without them the box shows
the chosen name and the hidden id is sent unchanged.

## Why it works this way

- **Its own list, not the browser's.** A native datalist is not enlarged
  when the page is zoomed in any desktop browser, is not shown by Firefox on
  Android, is lost by Chrome on Android turned sideways, and cannot be opened
  by voice control on an iPad; its options cannot be made 44px either. So
  the script draws a listbox and follows the APG list autocomplete
  ([Adrian Roselli, under-engineered comboboxes](https://adrianroselli.com/2023/06/under-engineered-comboboxen.html),
  [APG list autocomplete](https://www.w3.org/WAI/ARIA/apg/patterns/combobox/examples/combobox-autocomplete-list/)).
- **Said as it happens.** How many were found, that none were, that two
  letters are needed, or that the search failed, in the words GOV.UK's
  autocomplete uses ([GOV.UK accessible autocomplete](https://github.com/alphagov/accessible-autocomplete)).
- **No wrong record sent quietly.** Changing the name unchooses the record;
  a name that is none of them is said to be so beside the box
  ([WCAG 3.3.1](https://www.w3.org/WAI/WCAG22/Understanding/error-identification)).
- **Only records of the kind asked for** are searched, so the fifty notes
  that also match never crowd out the person being looked for
  ([NN/g on scoped search](https://www.nngroup.com/articles/scoped-search/)).
- **Two records with one title** are told apart by a line from each.

Not done, and why: typing ahead into the box (screen reader users heard only
the added letters); choosing the first match when the box is left (matches
here are loose, and it could save the wrong record); a Show all button
(meaningless over thousands).

