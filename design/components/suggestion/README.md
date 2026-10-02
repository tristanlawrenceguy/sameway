# suggestion

A change to someone's writing that was suggested and not made. It says who
suggested it and why, then shows the words where they are: the sentence as
it is now, with what would change highlighted, and as it would read with
the change. Accept makes the change; Decline sets it aside.

The assistant and agents make these with the `suggest_edits` tool when they
are asked to improve words a person wrote. Each waits on the page of the
record it is about, for whoever may change that record.

## Why it works this way

- **Their words stay theirs.** Rewriting in place leaves the writer to find
  what changed; a suggestion shows it, and nothing changes until they say.
- **One answer per change.** Each is accepted or declined on its own, as in
  Google Docs' suggesting mode and Word's track changes, so one good change
  does not carry a bad one with it. Only fixes and formatting can be
  accepted all at once
  ([Google Docs](https://support.google.com/docs/answer/6033474)).
- **Accepting is an ordinary change**: logged as the person's, and undone
  from the activity log like any other.
- **Who suggested it, in words.** "Suggested by the assistant", or an agent
  by the name the activity log has for it, beside the edge's colour. The
  colour alone said it before, and in forced colours it said nothing.
  GitHub likewise names each suggester
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html),
  [GitHub suggested changes](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/incorporating-feedback-in-your-pull-request)).
- **Where the change starts and ends is heard, once.** A screen reader
  hears "Change starts:" before the highlighted words and "change ends"
  after them. It is a plain span, not `mark`, `ins` or `del`: screen
  readers say those differently, and some never say where they end
  ([Vispero](https://vispero.com/resources/screen-readers-support-for-text-level-html-semantics/),
  [Adrian Roselli](https://adrianroselli.com/2017/12/tweaking-text-level-styles.html)).
- **Not by colour alone.** The highlight has an outline as well as a tint,
  and in forced colours it takes the system's highlight colours. Captions
  say which side is Now and which is With the change.
- **Shown as it reads.** The writing is formatted, never Markdown, and a
  change to the formatting alone is said in words: "Make it a list".
- **Outdated says so.** When the words it would change have changed since,
  it does not guess where it was meant to go: it says it no longer fits and
  offers only to set it aside.
- **Fits at 320px.** The two sides stand side by side where there is room
  and one above the other where there is not, never wider than the screen
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html)).

Not done, and why: the ARIA 1.3 `suggestion`, `insertion` and `deletion`
roles (a draft that screen readers barely support; the words already say
it); striking through what goes and underlining what comes, as word
processors do (the two sides already show before and after, and a
formatting change takes no words out); keys to jump to the next suggestion,
as Google Docs has (each kind has a heading, and the next suggestion's
answers are the next Tab stops); Decline all (declining is cheap one at a time, and a
whole set of suggestions turned away at once is rarely what someone
means).
