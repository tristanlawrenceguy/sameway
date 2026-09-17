# Working in this workspace as an agent

This folder is a Sameway workspace: content types in `schema/`, every record
mirrored as Markdown in `content/`, files people added in `files/`, and the
live store in `data.db`. People use it through the pages; you use the same
things through three surfaces that all read and write the same records.

## The quickest way in: MCP

Run `sameway connect <tool>` here and it prints the exact configuration for
your tool (claude-code, claude-desktop, cursor, windsurf, vscode, codex), or
writes it with `--write`. You then have the assistant's own tools plus
`describe`, `look` and `get_record`: make records, place blocks on the canvas,
find and read anything, add a field or a type. Everything you do lands in the
activity log with your name on it and can be undone by the person.

A client elsewhere reaches the same server at `POST /mcp` once `sameway serve`
is running and the environment variable named by `mcp.token_env` in
`workspace.yaml` (`SAMEWAY_MCP_TOKEN` by default) is set; send it as
`Authorization: Bearer <token>`.

## The API and the command line

`GET /api/describe` says everything: types with their fields, components with
their props, the assistant's tools, every route. `GET /api/<type>`,
`POST /api/<type>`, `PUT /api/<type>/<id>` and `DELETE` do what they say;
`?where=done=false&where=due<=+7d&order=due` picks records the way a
collection block does. `POST /api/chat` asks the assistant. The command line
mirrors it: `sameway note list --where status=draft --json`,
`sameway task create --set title="Order compost" --set due=2026-10-01`.

## What to keep in mind

- Content is not the canvas. A note is a record on `/t/note`; a card with its
  words copied in is not a note. Show a record on the canvas with a record
  block, or many with a collection block.
- Say where things are, as their page path: `/t/task/<id>`.
- A reply that names a page which does not exist made nothing. Use the tools;
  the receipt under a reply is what happened.
- Adding a field or a type changes the workspace for everyone and takes
  nothing away; removing anything is the person's call.
- `content/` is the workspace's memory in plain files: after a `git pull`,
  `sameway import` reads it back into the store.
