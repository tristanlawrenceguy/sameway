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

## Why it works this way

- **What it takes, said first.** The kinds of file that are read, that any
  other is kept as it is, and the 4 GB limit are under the label, before
  anyone chooses ([GOV.UK file upload](https://design-system.service.gov.uk/components/file-upload/),
  [WCAG 3.3.4](https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-all.html)).
- **A problem in words above the field**, marked with a bar and as
  invalid, in the same words on the page and from the server: select a
  file, too big, empty ([GOV.UK error message](https://design-system.service.gov.uk/components/error-message/)).
- **Too big is caught before it is sent**, not after the whole file has
  gone.
- **Adding the file says so** while it goes, and cannot be sent twice.
- **Every upload form works the same**, the list's and one placed on the
  canvas.
- **The browser's own file field**, which works without scripts, takes a
  file dropped on it, and keeps each system's chooser.

Not done, and why: a drop zone of our own (the field already takes drops,
and two actions in one control confuse screen readers); a strict list of
accepted types (any file is worth keeping, and the chooser greys others
out without saying why); several files at once (one at a time is what the
list, the chat and import share).
