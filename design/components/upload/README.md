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
- **A big file shows how much has gone.** Past 10 MB the form is sent by
  the page so a bar can show it going, with "Sending the file: 1.2 GB of
  3.4 GB sent." in the status each quarter, not each step, so a screen
  reader hears it now and then. Past ten seconds a wait wants a measure,
  not a spinner ([NN/g progress indicators](https://www.nngroup.com/articles/progress-indicators/),
  [WCAG 4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html)).
  It ends where the form would have, with what the server said there; a
  lost connection says so and the form can be sent again ("could not be
  uploaded, try again", [GOV.UK](https://design-system.service.gov.uk/components/file-upload/)).
- **A file that cannot be read says what happened and what to do**, never
  the reader's own words ("zip: not a valid zip file"): locked with a
  password (save a copy without it), damaged or not really what its name
  says (save a fresh copy), an older .doc, .xls or .ppt (save it as the
  newer kind), pages that are pictures, an ending Sameway does not read.
  The original still opens ([NN/g error messages](https://www.nngroup.com/articles/error-message-guidelines/),
  [WCAG 3.3.3](https://www.w3.org/WAI/WCAG22/Understanding/error-suggestion.html)).
  An agent reads the same words in the file's `note`, and the assistant
  hears them with the file.
- **Read as what it is.** A PDF named .docx is read as a PDF; text saved
  as Notepad's UTF-16 or Excel's Windows-1252 CSV reads with its é and £.
- **What leaves the computer.** The original stays in the workspace's
  `files/` folder. The assistant's model is sent a file's words (the
  first 8,000 characters) only with a message it is attached to or when
  it reads the file, and a picture, made smaller, only to a model that
  can see.

Not done, and why: a drop zone of our own (the field already takes drops,
and two actions in one control confuse screen readers); a strict list of
accepted types (any file is worth keeping, and the chooser greys others
out without saying why); several files at once (one at a time is what the
list, the chat and import share). A bar for a small file (it is gone
before a bar would mean anything, and the form is sent as it always was);
a bar in the chat's Attach a file (its turn already streams through the
page, and the files page is where a big file goes); a password field for a
locked PDF (the password would sit beside the file; saving a copy without
it keeps the choice with the person).
