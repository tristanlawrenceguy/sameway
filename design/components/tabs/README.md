# tabs

A row of sibling pages, such as the canvases, with the one a person is on
marked. Each tab is a link to its page, so it works without a script and
behaves like every other link: Tab to it, Enter to go.

These are not an ARIA tablist. A tablist shows and hides panels on one page
and needs arrow keys a person has to know about; these go to pages. Show the
bar only when there are at least two places: one tab says nothing.

## Why it works this way

- **Links, not ARIA tabs.** Each tab is a page of its own, so the bar is a
  navigation of links with the one here marked; the ARIA tabs pattern is for
  panels within one page ([GOV.UK tabs](https://design-system.service.gov.uk/components/tabs/),
  [APG tabs](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/)).
- **Each page is called by its tab**, Home or Garden, in the window title
  and heading, so arriving on one is heard
  ([WCAG 2.4.2](https://www.w3.org/WAI/WCAG22/Understanding/page-titled.html)).
- **The other tabs look pressable** without a pointer: underlined, as links
  on their own are; the one here has a bar, weight and the text colour, and
  keeps its bar alone in forced colours
  ([NN/g, tabs used right](https://www.nngroup.com/articles/tabs-used-right/)).
- **No two tabs with one name**, and names of a word or two.
- **A tab that has gone leads Home** after the turn that removed it, and
  an address of one that is not there gets the site's own page saying so.
- **Every tab in sight**: they wrap onto a second line rather than scroll
  out of view.

Not done, and why: a close or add button on each tab (removing a tab takes
its blocks and is asked first; the assistant adds them); a menu button on a
phone (it hides places, for a handful of tabs).
