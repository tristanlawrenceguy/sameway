# Content

Every record lives here too, as Markdown with YAML front matter: one folder
per content type, one file per record, named by its id. The first `text` or
`markdown` field of the type is the body; every other field, and when the
record was created and updated, is in the front matter. Blocks and tabs are
here as well, because the page is content; the conversation, its questions
and the activity log are history and stay in `data.db`.

The files are written as records change, so this folder is always the
portable form of the workspace. Share it with git. After a pull, `sameway
import` reads it back into the database, logging each change so it can be
undone; `sameway export` rewrites the folder from the database.
