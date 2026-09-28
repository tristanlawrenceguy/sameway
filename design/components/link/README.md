# link

Use a link to go somewhere: another page, a record, an anchor. Use `button`
for an action that stays on the page. The label should make sense read on its
own, because screen reader users often list links out of context. Set
`current` on the link that points at the page being viewed.

Never open a new tab. Where one must, say so in the visible words: "Open
(new tab)". An address shown as a link reads as where it goes
(example.com/guide), not with its https://.

## Why it works this way

- **Underlined, always.** The link colour is 7:1 on the page but barely
  different from the words around it, so the underline says it is a link;
  a button-shaped link is underlined too, since a touch screen has no hover
  to show its shape ([WCAG F73](https://www.w3.org/WAI/WCAG22/Techniques/failures/F73),
  [GOV.UK links](https://design-system.service.gov.uk/styles/links/)).
- **Here, not only by colour.** The current page is bold with a thick
  underline, in the navigation with a bar at its edge
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **A hit area that fills its line.** A link in a sentence can be pressed
  anywhere on its line, never on the line above or the next link; links on
  their own are 44px ([WCAG 2.5.5](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html)).
- **No surprise tabs.** A new tab is said in words
  ([WCAG 3.2.5](https://www.w3.org/WAI/WCAG22/Understanding/change-on-request.html),
  [Adrian Roselli](https://adrianroselli.com/2020/02/link-targets-and-3-2-5.html)).

Not done, and why: a visited colour (these are the app's own places, which
change; a second blue would need its own 7:1 and a cue beyond colour); an
external-link icon (GOV.UK advises against it, and it says where, not how).
