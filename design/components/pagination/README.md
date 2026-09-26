# pagination

The way through a long list a page at a time: Previous, the pages by number,
and Next. Every page is a plain link, so it works without scripts and each
page has its own address a person can keep or share.

Give it the address of the list (`href`), which may carry its own query such
as a search; the page goes on the end as `?page=2`, and the first page has
none. It shows the first and last pages and one either side of this one, and a single page rather than a gap standing for it, with
a gap for the rest.

Say how many there are in all above the list, such as "230 notes"; this only
moves between them. Use it when a list could grow past what reads well on one
page, around 50 rows.

## Why it works this way

- **The window title says the page**: "Search: fern, 25 results, page 2 of
  2", the first thing a screen reader says as the page arrives, so moving on
  is heard ([GOV.UK pagination](https://design-system.service.gov.uk/components/pagination/),
  [WCAG 2.4.2](https://www.w3.org/WAI/WCAG22/Understanding/page-titled.html)).
- **Numbered results count on**: page 2 starts at 21, not at 1 again.
- **Never more than seven** numbers and gaps, and a gap always stands for
  two pages or more ([USWDS pagination](https://designsystem.digital.gov/components/pagination/)).
- **On a phone** Previous and Next share a line and the numbers have their
  own, rather than wrapping wherever they fall.
- **Each number is a 44px target** that shows it can be pressed; the current
  one is filled and bold, not only coloured.

Not done, and why: infinite scroll or Load more (it breaks Back and
bookmarks and keeps a keyboard from the footer); a Go to page box (a form
for a rare need).

