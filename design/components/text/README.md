# text

Use text for prose: explanations, notes, answers. Separate paragraphs with a
blank line in `content`. For a short label with an icon or emphasis, use the
component that fits (alert, heading, card) rather than styling text.

## Why it works this way

- **Headings keep the page's outline.** The shallowest heading written is
  the section, whatever it was written as, and none skips a level: the
  assistant's ## Plan and ### Step become the section and its part
  ([WAI headings](https://www.w3.org/WAI/tutorials/page-structure/headings/)).
- **Lines a person can follow**: 68 characters at most, a line and a half
  apart, never justified; the Wide spacing setting sets paragraphs a line
  and a half apart and gives headings the wider spacing too
  ([WCAG 1.4.8](https://www.w3.org/WAI/WCAG22/Understanding/visual-presentation.html),
  [WCAG 1.4.12](https://www.w3.org/WAI/WCAG22/Understanding/text-spacing.html)).
- **Quieter by colour only**, at 7:1: muted words are not made smaller too
  ([GOV.UK type scale](https://design-system.service.gov.uk/styles/type-scale/)).
- **A long address wraps**, never widening the page on a phone.
- **Editing lets a keyboard out.** Tab leaves the text even in a list;
  indenting is Ctrl+] and Ctrl+[
  ([WCAG 2.1.2](https://www.w3.org/WAI/WCAG22/Understanding/no-keyboard-trap.html)).
- **The Markdown switch keeps its name**, pressed or not, and says so if it
  could not switch ([APG button](https://www.w3.org/WAI/ARIA/apg/patterns/button/)).

Not done, and why: a wider gap between paragraphs for everyone (the Wide
setting gives it to whoever wants it, and the rest keep a calmer rhythm).
