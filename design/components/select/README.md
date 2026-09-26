# select

Use a select for one choice from a fixed set, such as a status, a kind or a
period. It is one component whatever it looks like, so the value it stores
and the way it is described are the same everywhere.

With five options or fewer it shows them as radio buttons, all in view, in a
fieldset whose legend is the question: a person sees every answer at once and
chooses with one press. With more it is a dropdown, so a long list does not
fill the page. Set `as` to `radios` or `dropdown` when the author knows
better, such as a dropdown in a table cell where radios would not fit.

Both are native controls on purpose: every screen reader and browser already
knows how to drive them. It does not change with the device; a phone shows a
native dropdown as a full-screen picker anyway. For a record out of hundreds,
use a lookup; for several choices at once, checkboxes.

With no answer yet, a dropdown starts on "Choose one" and radios start with
none chosen: nothing is chosen for the person, and a required one is
refused until something is. List the answers in alphabetical order unless
they have an order of their own, such as sizes or days.

## Why it works this way

- **Radios for five or fewer, a dropdown for more, a lookup past fifteen.**
  Radios show every answer; a dropdown hides them; a list longer than
  fifteen is read more quickly by typing part of a name
  ([NN/g on listboxes and dropdowns](https://www.nngroup.com/articles/listbox-dropdown/),
  [USWDS select](https://designsystem.digital.gov/components/select/)).
- **Nothing chosen for the person.** A dropdown with no answer showed the
  first one, and saving wrote it; it now starts on "Choose one", which is
  also what lets required refuse an empty answer
  ([HTML, the placeholder option](https://html.spec.whatwg.org/multipage/form-elements.html#placeholder-label-option),
  [GOV.UK radios](https://design-system.service.gov.uk/components/radios/)).
- **A record's choices in order of their names**, not the order they were
  made.
- **A problem shows as a bar at the edge** as well as in words.
- **Text spacing a person sets reaches the box**
  ([Adrian Roselli, under-engineered selects](https://adrianroselli.com/2021/03/under-engineered-select-menus.html)).
- **Each radio's whole row can be pressed**, the box the size of a checkbox.

Not done, and why: a drawn chevron or a custom list (the browser's own works
with zoom, forced colours, phones and voice control).
