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
