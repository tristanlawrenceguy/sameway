# meter

How far along a known range something is, as a thin bar: 3 of 8 glasses, 40
of 60 hours, a budget part spent. Give it a label that names the thing and
words (`text`) that say the amount the way a person would, such as "3 of 8
glasses"; a screen reader reads the words, not a bare number.

`state` colours it: `going` in the list colour, `met` in success green, `over`
in the warning colour for a limit gone past. The words should say it too,
because colour alone does not.

A value past `max` fills the bar and is read as full; say how far over in the
words. For work in progress with no known end, use a status instead.

## Why it works this way

- **What it measures, and how much, where it can be seen.** A bar alone says
  neither; the words sit above it unless they are already beside it, as in
  the tracker (`words: false`)
  ([APG meter](https://www.w3.org/WAI/ARIA/apg/patterns/meter/),
  [NN/g on progress indicators](https://www.nngroup.com/articles/progress-indicators/)).
- **A track that can be seen**, with an edge at 3:1, so how far it could go
  shows even when it is empty
  ([WCAG 1.4.11](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html)).
- **Over is striped**, not only a different colour, and the stripes start
  at the target ([WCAG G111](https://www.w3.org/WAI/WCAG22/Techniques/general/G111)).
- **Green means reached.** A limit kept within stays in the list colour;
  only a target reached is green, and a limit gone past is striped warning.
- **Read in words**: 5 of 8 glasses, not a percentage.

Not done, and why: the native meter element (styling it is not the same in
every browser, and styled ones lose their name in some screen readers); a
percentage (meaningless for a limit, and for going over).
