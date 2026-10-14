# Refusals

How Sameway says no to an agent or a person: the JSON API,
`POST /api/tools`, MCP, the assistant's tools, the CLI and the pages. The
words are in `internal/records/refused.go` (a type or a record not there,
a word not taken), the API's in `internal/server/api/api_refusals.go` and
`api_params.go`, the tools' in `internal/chat/unknown_args.go`, the error
shape in `internal/web/json.go`. Every code the API sends is named in
describe's `errors` entry (`tools/check/errcodes.go`). Tests:
`internal/server/api/refusals_test.go`, `internal/server/args_not_taken_test.go`,
`internal/mcp/not_taken_test.go`.

## Why it works this way

- **Nothing sent is silently ignored.** A query word a read does not
  take, an argument a tool does not take, a field a type does not have:
  each is refused, never answered as if it were not there. An answer
  that ignored `since` looked filtered and was not (crew finding 0742).
  Stripe answers "Received unknown parameter", and JSON:API requires 400
  for a query parameter the server does not support
  ([JSON:API, query parameters](https://jsonapi.org/format/#query-parameters)).
- **A refusal says what was wrong, what is taken, and the one likely
  meant.** "GET /api/changes does not take snce (did you mean since?). It
  takes since, limit, wait". Anthropic's guidance on tools for agents:
  error responses should say "specific and actionable improvements" and
  can show correctly formatted input
  ([Writing tools for agents](https://www.anthropic.com/engineering/writing-tools-for-agents)).
  RFC 9457's `detail` "ought to focus on helping the client correct the
  problem" ([RFC 9457, 3.1.4](https://www.rfc-editor.org/rfc/rfc9457.html#name-detail)),
  so Go's own words (`strconv.Atoi: parsing "abc"`) never reach a caller.
- **Where to read more is a link.** Beside `message`, an API refusal
  has `describe`: the part of `GET /api/describe` that says what is
  taken (the type, the routes, the tools), as Stripe's `doc_url`,
  GitHub's `documentation_url` and JSON:API's `links.about` do.
- **The same mistake is said in the same words everywhere.** A type not
  there is "there is no content type tasks (did you mean task?); the
  workspace has …" on the API, in the tools, on a block's page and in the
  CLI; a record not there is "no note with id x". Each way in adds only
  its own next step: `GET /api/note`, `find_records`, `sameway note list`.
  Google's AIP-193 keeps a reason the same for the same error
  ([AIP-193](https://google.aip.dev/193)).
- **The status is the kind of mistake.** 400 a request not understood (a
  word not taken, a number that is not one, JSON that is not JSON); 404 a
  type, a record, a route or a tool not there; 405 the log, with `Allow:
  GET` as RFC 9110 requires
  ([RFC 9110, 15.5.6](https://www.rfc-editor.org/rfc/rfc9110.html#name-405-method-not-allowed));
  412 a stale `If-Match`; 422 fields or a tool's arguments understood but
  refused. A type not there was 400 and a record not there 404: one
  mistake, two answers.
- **A stable code, words for the occurrence.** `code` is what a program
  branches on and never changes; `message` may say more over time. That
  is AIP-193's reason and message, Stripe's `code` and `message`, and
  JSON:API's `code` and `detail`.
- **What a person agrees to is not a caller's to say.** `agreed` is the
  person's Yes, carried only by a question's answer. Sent by a caller,
  `change_field` used to take it and delete a field without asking; it is
  now an argument the tool does not take.
- **Several writes, or several undos, are all or none, put back one
  way.** `records.WriteAll` (several writes) and `Book.UndoEach` (several
  undos) are two jobs, but when one of them is refused both put back what
  they did by running it backwards: `Book.putBack`, once. They had two
  copies of it.

## Not done, and why

- **No `application/problem+json`.** The shape `{"error": {"code",
  "message", "fields"}}` is in describe, the CLI and every agent written
  against it; RFC 9457's members map onto it (code for type, message for
  detail, the HTTP status as is). Changing the media type would break
  them for no new information.
- **A tool's refusal over `POST /api/tools` is 422 whatever it says.**
  The tool's words are the same ones the model and MCP read, and MCP has
  no status; telling a missing record from a refused field there would
  mean every tool saying which, for a status the words already carry.
- **Unknown query words on a page are not refused.** A page is read by a
  person, who did not type its address; a stray word there (`?utm_source`)
  is not a question asked.
