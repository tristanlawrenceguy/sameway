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
