# crumbs

The way back from where a person is. A record's page leads with the list it
belongs to; a deeper page leads with each place above it, outermost first.

Give `current` only when nothing else names the page: on a record's page the
heading below already says the record's name, and saying it twice makes a
screen reader say it twice, so a trail usually ends with the place above the
page, not the page. The chevrons between places are drawn with borders, not
characters, so nothing is read out for them. The landmark is named
Breadcrumb, the name screen reader users know; "You are here" would say they
are on the place it leads back to. Give a place its `dot` to show which of the person's lists it
is, the same colour the list has in the navigation.

## Why it works this way

- **A chevron nothing reads.** A character in CSS generated content is read
  out by screen readers ("single right-pointing angle quotation mark"), so
  the chevron is drawn with two borders, which also keep their shape in
  forced colours
  ([APG breadcrumb](https://www.w3.org/WAI/ARIA/apg/patterns/breadcrumb/)).
- **Named Breadcrumb**, the name screen reader users know it by.
- **The page not said twice.** A trail usually ends above the page, since
  the heading under it names the page
  ([GOV.UK breadcrumbs](https://design-system.service.gov.uk/components/breadcrumbs/)).
- **A dot on the link**, part of what is pressed, so each place keeps its
  chevron.
- **Long names wrap** rather than running off a phone
  ([WCAG 1.4.10](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html)).

Not done, and why: the name "You are here" (it says the person is on the
place the trail leads back to); a trail in place of navigation (it shows
where a page sits, not everything there is;
[NN/g breadcrumbs](https://www.nngroup.com/articles/breadcrumbs/)).
