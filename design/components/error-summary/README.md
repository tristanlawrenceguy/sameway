# error-summary

When a form is sent and some of its answers cannot be taken, say so at the top,
once, as a list: each problem in plain words with how to fix it, linking to the
field it is about. A person who sent a long form does not then have to hunt
for what went wrong, and a screen reader hears it as an alert when it arrives.

Name the field with `field`; the link goes to it, and with scripts pressing it
moves focus there, opening the record's editor first when it is closed. Each
field it names is marked invalid, with the problem beside it in words, above
the input, as a field's own error is; a red edge alone would say it only in
colour. Focus goes to the summary once, when the problems arrive, so the list
is heard first and each link leads on. Use `href` for a problem that is
somewhere else; one with neither is plain words.

A page showing it has a window title that starts "Error:", the first thing a
screen reader says when the page arrives.

## Why it works this way

- **The problem beside its field** in words, above the input, and tied to
  it, since a red edge alone says it only in colour
  ([GOV.UK error message](https://design-system.service.gov.uk/components/error-message/),
  [WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **Focus on the list, once.** The whole list is heard first and each link
  leads on; a field does not hear the whole summary as its description
  ([GOV.UK error summary](https://design-system.service.gov.uk/components/error-summary/),
  [WAI forms notifications](https://www.w3.org/WAI/tutorials/forms/notifications/)).
- **Error: first** in the window title, the first thing a screen reader
  says, and the summary's title carries the danger alert's mark and word
  ([WCAG 2.4.2](https://www.w3.org/WAI/WCAG22/Understanding/page-titled.html)).
- **Every problem listed**, in plain words, even one about a field the
  type does not have; one with nowhere to lead is not a link
  ([WCAG 3.3.1](https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html),
  [NN/g error messages](https://www.nngroup.com/articles/error-message-guidelines/)).
- **Each problem is a 44px target** that does not overlap the next
  ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).

Not done, and why: moving focus to the first bad field (the person would
not learn how many problems there are); a summary that closes itself
(the problems are still there to fix).
