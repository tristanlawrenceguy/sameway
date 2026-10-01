# suggestion

A change to someone's writing that was suggested and not made. It says why,
then shows the words where they are: what would go struck through, what
would come underlined, a few words either side so it reads as the sentence
it would become. Accept makes the change; Decline sets it aside.

The assistant and agents make these with the `suggest_edits` tool when they
are asked to improve words a person wrote. Each waits on the page of the
record it is about, for whoever may change that record.

## Why it works this way

- **Their words stay theirs.** Rewriting in place leaves the writer to find
  what changed; a suggestion shows it, and nothing changes until they say.
- **One answer per change.** Each is accepted or declined on its own, as in
  a word processor's suggesting mode, so one good change does not carry a
  bad one with it.
- **Accepting is an ordinary change**: logged as the person's, and undone
  from the activity log like any other.
- **Not by colour alone.** Strike and underline carry what goes and what
  comes, and a screen reader hears "Take out" and "Put in"
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color)).
- **Outdated says so.** When the words it would change have changed since,
  it does not guess where it was meant to go: it says it no longer fits and
  offers only to set it aside.
