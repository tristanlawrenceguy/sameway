# status

Use a status to say what the system is doing right now. It is a live region,
so the text is announced when it changes, and it carries `data-state` so an
agent can read it. Give it an `id` and point a form at it with
`data-busy-target` and `data-busy-message`; the enhancement script then
flips it to "working" while the request is in flight, and the server's
response sets the final state. Without JavaScript the form still works; the
page simply reloads with the result.
