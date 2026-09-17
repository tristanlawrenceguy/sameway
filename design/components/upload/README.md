# upload

Adds a file. A real multipart form with a labelled file field and one
button, so it works without a script and every platform's own chooser
and screen reader support come for free. The file becomes a record of the
`file` type: the original is kept under the workspace's `files/` folder,
and its contents are read into Markdown that the page renders with
structure, search finds by its words, and the assistant can read. Formats
Go can read are handled in the binary; a converter named in
`workspace.yaml` handles the rest, such as docling for PDFs.

It lives on the files page by default, and can be placed anywhere the
assistant puts blocks, at compact or icon size.
