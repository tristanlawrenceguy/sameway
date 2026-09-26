# status

Use a status to say what the system is doing right now. It is a live region,
so the text is announced when it changes, and it carries `data-state` so an
agent can read it. Give it an `id` and point a form at it with
`data-busy-target` and `data-busy-message`; the enhancement script then
flips it to "working" while the request is in flight, and the server's
response sets the final state. Without JavaScript the form still works; the
page simply reloads with the result.

## Why it works this way

- **One way to change it.** Its state, its look, its words and what is read
  after them change together, in the region already on the page: a region
  put in whole is not read out, and words changed without their state
  showed a failure beside a success's tick
  ([WCAG ARIA22](https://www.w3.org/WAI/WCAG22/Techniques/aria/ARIA22),
  [Scott O'Hara, are we live](https://www.scottohara.me/blog/2022/02/05/are-we-live.html)).
- **Always polite.** A person who started the wait is listening for its end;
  an interruption, and a region whose politeness changes, are poorly
  supported and disruptive
  ([Sara Soueidan on live regions](https://www.sarasoueidan.com/blog/accessible-notifications-with-aria-live-regions-part-1/)).
- **A long wait is said once to be still going**, after fifteen seconds,
  never again ([NN/g on progress indicators](https://www.nngroup.com/articles/progress-indicators/)).
- **A file being read says how it ended**, failure included, and the page
  follows when it does, instead of working for ever.
- **A page brought back with Back** asks the server how the request ended
  rather than showing working for ever.
- **Each state has its own shape**: an open ring, a tick, a mark for an
  error, never colour alone.

Not done, and why: a percentage (the length of the work is not known); each
step said as it happens (too much to hear); a turning spinner (the words say
it, and nothing moves on its own for long).
