# fields

A thing's facts, each with its name: Status Draft, Due Fri 2 Oct, Project
Garden. A record's page shows what its heading and chips have not already
said this way.

A value can be words, a link to what it names (`href`), or structured text
(`markdown`), which keeps its headings and lists. Give a value its `prop`
(and its stored `source`, `kind` or `options` when they differ from what
reads) and the inline editor knows which field it is; without them it is only
read.

Set `compact` for facts inside a card, a row or a record block, where they
are a detail rather than the page: smaller and closer, without hairlines.
Components that show facts (record, collection) render this component
through the `fields` template helper rather than a list of their own.

Leave empty facts out rather than showing a name with nothing beside it.
