# problem

What a block says when it cannot show what it was asked for: a list of a
type the workspace does not have, a chart grouped by a missing field, a
calendar of records with no date, habits with a tag none of them carry.

"This calendar cannot be shown as it is set up. Ask the assistant to fix
it." comes first, in plain words, with who can fix it. What is wrong, in
the workspace's own names, is under What is wrong.

Collection, chart, calendar and tracker all use it, so a block set up
wrong reads the same wherever it is.

Name what cannot be shown in the singular, as a person says it (This
list, This list of habits), since the sentence goes on "as it is set up".

It is not an empty state and not an alert. Nothing matching is an empty
state: the block works, and there is nothing to show. A failed action is
an alert: something the person just did did not happen. A problem is a
block that is wrong before anyone does anything, and stays until the
assistant changes it.

## Why it works this way

- **Never nothing.** A block that renders empty reads as "there is nothing
  here", which is a different thing, and a person acts on it
  ([WCAG 3.3.1](https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html),
  [NN/g on empty states](https://www.nngroup.com/articles/empty-state-interface-design/)).
- **Who can fix it, then what.** Most people need to know it is not their
  doing and who can mend it; whoever mends it needs the exact names
  ([W3C COGA](https://www.w3.org/TR/coga-usable/),
  [NN/g error-message guidelines](https://www.nngroup.com/articles/error-message-guidelines/)).
- **Not their fault, no apology, no jargon.** "Cannot be shown as it is
  set up" puts it on the block, not the person, and says no "invalid",
  "sorry" or "oops"
  ([GOV.UK error messages](https://design-system.service.gov.uk/components/error-message/),
  [Microsoft writing style](https://learn.microsoft.com/en-us/windows/apps/design/style/writing-style)).
- **Technical detail on request.** The type and field names are for
  whoever fixes it, so they are a press away, not in the first sentence
  ([NN/g error-message rubric](https://www.nngroup.com/articles/error-messages-scoring-rubric/)).
- **A native disclosure with the twisty**, so the detail opens with no
  script, and What is wrong looks like every other fold that opens; it
  had lost its twisty, and read as grey words
  ([GOV.UK details](https://design-system.service.gov.uk/components/details/),
  [NN/g accordion icons](https://www.nngroup.com/articles/accordion-icons/)).
- **In the text colour.** A quiet empty line is muted and sits in the
  same place; a block set up wrong must not look like one with nothing in
  it. No red: nothing the person did is wrong
  ([GOV.UK service unavailable pages](https://design-system.service.gov.uk/patterns/service-unavailable-pages/)).
- **Read as it sits, not announced.** It arrives with the page, where a
  live region is not heard, so it has no live role
  ([APG alert pattern](https://www.w3.org/WAI/ARIA/apg/patterns/alert/)).
- **Word for word, with a capital on screen.** The detail is the server's
  own words for an agent to read; the style starts it with a capital, as
  a sentence should, without changing them. Long names break and lines
  stop at the measure, so it reads at 320px
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html),
  [WCAG 1.4.8](https://www.w3.org/WAI/WCAG22/Understanding/visual-presentation.html)).

Not done, and why: "Ask the assistant" as a link with the question ready
(the problem does not know which canvas or block it is in, so the
question could name neither, and a canvas without a chat has no box to
put it in); a different sentence for someone who may only look (they are
told who can change things when they open the assistant); a warning mark
or tint (it would say the person has something to fix, which they do
not); a live role (not heard on page load); the detail open on paper (the
disclosure's print script is its own, and a printed canvas with a broken
block is rare); naming the block in What is wrong (the sentence just
before it says which).
