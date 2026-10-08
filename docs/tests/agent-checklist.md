# Tests: what outside agents need, the checklist

Beside the rules for people, the 22 checks from the agent accessibility
audit (2026-09-28): a browser agent finds controls by role and name in the
accessibility tree, a screenshot agent sees only what is drawn, a tool agent
reads MCP and the API. Where each is checked: `agent` is
`tools/a11y-runner/agent.mjs`, `site` is `site.mjs`, `look` is `/api/look`'s
problems, `go` a Go test; `known` is reported by agent.mjs but fails only
with `--strict` until its fix lands; `review` and `not yet` are yours to hold.

| # | Check | Where |
|---|---|---|
| | **Perceive** | |
| 1 | Every control named, in the browser's own tree | agent, look |
| 2 | No two controls of one role and name within a landmark (records alike are told apart by their due day, when added, or id); links of one name go to one place | agent, look, site, go `names_apart_test.go` |
| 3 | What a control shows is in its name; hidden words add to it, never repeat it | agent, run.mjs |
| 4 | No operable control below 0.35 opacity at rest | agent (known) |
| 5 | The document scrolls as one page, so a full-page screenshot has it all | not yet |
| 6 | No meaning only in a canvas, a hover or a drag | review |
| | **Understand** | |
| 7 | One ariaSnapshot golden per page type | not yet |
| 8 | No glyphs in names; no counts run into words ("This week 1 28 Sep") | agent (glyphs); go `machine_words_test.go` (counts and any two values with nothing between them) |
| 9 | A manifest's `machine` says how to find it by role and name, and that holds | go `contract_test.go` (selector); role-and-name not yet |
| 10 | look says what the browser's tree says | agent (look on the same pages); not compared yet |
| | **Operate** | |
| 11 | Everything works without scripts | go `pages_test.go`, review |
| 12 | After an in-place change the page equals a reload | agent (known) |
| 13 | An outcome stays, in a status or alert, names the record, offers Undo | agent (tick, undo), go `back_test.go`, `reversible_test.go` |
| 14 | Nothing is written until a deliberate press; a control that writes says so | review |
| | **Tools** | |
| 15 | Every MCP tool titled and annotated; read-only ones proved so by a store diff | `internal/mcp/annotations.go`; `TestEveryToolSaysWhatItIs`, `TestReadOnlyToolsChangeNothing` (the workspace on disk hashed before and after) |
| 16 | Every error is `isError` with the next step and the valid choices | go `internal/mcp/*_test.go` (partly) |
| 17 | List rows can be told apart and are paged | pages: agent, go; tool results not yet |
| 18 | `describe` has a small index under a budget | go `describe_index_test.go` (16 KB, block routes first), `internal/mcp/describe_size_test.go` (nothing but full past a client's limit), `prompt_budget_test.go` (the in-app prompt at 40 KB: a line per component and type, the rest in a refusal) |
| | **Trust** | |
| 19 | MCP and API writes are logged with the agent's name, from its key when it has one | go `agent_keys_test.go`, `writes_test.go` |
| 20 | Record text reaches outside agents marked as content, with who wrote it | `internal/chat/provenance.go` (written_by and untrusted on a record, a list, search, the changes feed, the tools and public MCP); go `untrusted_test.go`, `internal/mcp/untrusted_test.go` |
| 21 | Irreversible actions ask first; outward tools say openWorldHint | go `consent_test.go` (asking), `internal/mcp/annotations.go` (run_action and update_sameway say openWorldHint) |
| 22 | Writes are rate-limited per token | `internal/chat/pace.go` (60 changes a minute per key; reads and the owner unpaced); go `pace_test.go`, `agent_pace_test.go`, `internal/mcp/pace_test.go` |
