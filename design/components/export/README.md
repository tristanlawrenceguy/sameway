# export

Use export to let someone take something away as a file: one record, a
list as it is narrowed and sorted, a calendar, a recording's words, or the
whole workspace. It is a line saying what ("Download these 12 tasks") and
a link for each format that fits, most useful first:

| format | Link says | For |
|---|---|---|
| `csv` | Spreadsheet (CSV, 4 KB) | any list; opens in any spreadsheet, and comes back in through Import |
| `xlsx` | Excel (XLSX, 9 KB) | any list, with its header held and numbers as numbers |
| `vcf` | Contacts (vCard, 1 KB) | people, for an address book |
| `ics` | Calendar (ICS, 2 KB) | anything with a date, for a calendar app, which can also subscribe to the address |
| `md` | Text (Markdown, 1 KB) | one record: its title, its facts as a list, then its words |
| `html` | Web page (HTML) | one record, its page as a reader has it, whole in one file |
| `docx` | Word (DOCX, 9 KB) | one record, with Word's own title, headings and lists |
| `pdf` | Printable (PDF) | one record, its web page printed and tagged by the browser on the host |
| `zip` | Everything (ZIP, about 3 MB) | the whole workspace |
| `srt`, `txt` | Subtitles (SRT), Plain text (TXT) | a recording's words |

Offer only the formats that fit: no calendar for a list with no dates, no
contacts for tasks. The server works them out from the type
(`export.For`), so every list and record offers the same ones for the
same kind of thing.

Where it is: under a list, under a record, under a collection or a
calendar on its own page, and the whole workspace on Workspaces. A
published page offers it for what is published and nothing else.

## Why it works this way

- **The format and size in the link's words.** People decide whether to
  press before they press, on a phone's data too, and a screen reader
  listing links hears the same thing
  ([GOV.UK add links](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/writing-guidelines/add-links/),
  [NN/g on links to files](https://www.nngroup.com/articles/avoid-pdf-for-on-screen-reading/),
  [WCAG 2.4.4](https://www.w3.org/WAI/WCAG22/Understanding/link-purpose-in-context.html)).
- **A kind of file, not only an extension.** "Spreadsheet (CSV)" says what
  it opens in to someone who does not know what a CSV is, and the
  extension is still there for someone who does.
- **What it holds, said once.** The line above names the list, and each
  link's name ends with it in hidden words, so "Calendar (ICS, 2 KB),
  these 12 tasks" still makes sense out of context
  ([WCAG 2.4.9](https://www.w3.org/WAI/WCAG22/Understanding/link-purpose-link-only.html)),
  and begins with what is seen, for voice control
  ([WCAG 2.5.3](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html)).
- **What the page shows is what the file holds.** A list's links carry the
  page's own `where` and `order`, so the file is the narrowed list, not
  the whole type.
- **A plain link, no script.** Each is a GET that answers with the file
  and `Content-Disposition: attachment` with its name, `filename*` too for
  names beyond ASCII ([RFC 6266](https://www.rfc-editor.org/rfc/rfc6266));
  `download` asks the browser to keep it
  ([MDN](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/a)).
  Nothing downloads until pressed, and the page stays where it is.
- **Open formats, the ones things come in with**, so what goes out comes
  back: CSV ([RFC 4180](https://www.rfc-editor.org/rfc/rfc4180)) with a
  byte-order mark, because Excel reads UTF-8 without one as another
  encoding ([Microsoft](https://support.microsoft.com/en-us/excel/opening-csv-utf-8-files-correctly-in-excel));
  iCalendar ([RFC 5545](https://www.rfc-editor.org/rfc/rfc5545)), all day
  by the day with the end the day after, repeats as RRULE; vCard;
  Markdown; and the content folder's Markdown with front matter in
  everything. These are
  what the right to data portability asks for: structured, commonly used
  and machine-readable ([GDPR Art. 20](https://gdpr-info.eu/art-20-gdpr/),
  [ICO](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/individual-rights/individual-rights/right-to-data-portability/)).
- **A cell never runs as a formula.** A text that starts with `=`, `+`,
  `-`, `@`, a tab or a return goes out with a `'` before it, which a
  spreadsheet shows as text; Import takes it off again
  ([OWASP CSV injection](https://owasp.org/www-community/attacks/CSV_Injection)).
  Numbers are left as they are.
- **Everything at once, streamed.** The whole workspace is one ZIP made as
  it is sent, not a job to wait for and an email with a link: a personal
  workspace is small enough, and nothing is left on disk
  ([archive/zip](https://pkg.go.dev/archive/zip)). Its size is "about",
  from what goes in before it is compressed.
- **A size only when it is cheap and true.** A web page and a PDF are
  made by rendering and printing, so their links say none rather than
  a guess.

Not done, and why: JSON as a download (it is the API, at the same address
under /api, and linked from every page as its alternate; offering it twice
says it twice); a choice of what goes into everything (Google Takeout
has one because its account is dozens of products; here it is the
content folder, the files and a spreadsheet of each kind); an email when a large export is ready (a
job, a place to keep the file and a link that expires, for workspaces that
fit in a download); the conversation
and the activity log in everything (they are history, kept out of the
content folder for the same reason, and hold other people's words).
