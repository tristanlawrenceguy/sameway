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
