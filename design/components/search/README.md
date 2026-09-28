# search

One search over everything a person has. A form with a labelled field and
a button, marked as a search landmark, that sends its words to `/search`
as a GET, so the results page is a link that can be kept and shared. The
same search is behind `/api/search`, the assistant's search tool and
`sameway search`.

It belongs in the header, where a person reaches for it on every page; at
compact size it takes less room, and at icon size it is a glyph with its
name that opens the search page. In a bar the visible label is the button
text, and the field keeps its accessible name. Given a `type`, its results
open narrowed to that kind: put it on that kind's own place, labelled
Search tasks.

## Why it works this way

- **Reachable from every page**: Search is in the navigation, not only on
  the canvas ([NN/g, search visible and simple](https://www.nngroup.com/articles/search-visible-and-simple/)).
- **Forgiving**: case and accents do not matter, so cafe finds café, and a
  plural finds its one, so plumbers finds the plumber; when nothing has every
  word, what has some of them is shown, and said to be so
  ([W3C COGA, provide search](https://www.w3.org/TR/coga-usable/),
  [NN/g on site search](https://www.nngroup.com/articles/internal-website-search/)).
- **The words found are marked** in each result, bold on a wash, so a person
  sees why it is one.
- **Every result counted.** The page shows them a page at a time; it no
  longer stops at fifty and calls that the total.
- **A phone keyboard's key says Search**, the browser remembers past
  searches, and the box is wide enough for a few words
  ([USWDS search](https://designsystem.digital.gov/components/search/)).
- **At icon size it opens the search page**, where searching is.
- **Everything, then narrowed.** Search always covers everything; the
  results can then be narrowed to one kind, Notes or Tasks, by a row of
  links with how many each found, All first. They are links, so each
  narrowing is an address that works without a script, keeps the words and
  keeps its kind across pages; the one shown is marked `aria-current="page"`
  ([BBC GEL, filter and sort](https://bbc.github.io/gel/components/filter-and-sort/)).
  One kind is a filter, not facets: there is one thing to narrow by
  ([NN/g, filters vs facets](https://www.nngroup.com/articles/filters-vs-facets/)).
  The counts come from the one search. A kind with nothing found is not
  offered, and no row at all when only one kind was found, since it would
  be the same list as All. What the system keeps for itself, messages,
  activity, canvases, is never a kind to pick.
- **Said once, where it is heard.** A narrowed search says its kind in the
  heading and the window title, Search: seeds in tasks, 3 results, which a
  screen reader reads as the page arrives; a live region is for results
  that change in place, and these are a new page
  ([Scott O'Hara, dynamic results](https://www.scottohara.me/blog/2022/02/05/dynamic-results.html)).
- **Narrowed where it is asked, and the way out first.** Search from a
  kind's own pages, its list or one of its records, opens as Search tasks,
  with Searching tasks only and Search everything above the box; the results
  begin Showing tasks only, 3 of 7 found, and Search everything. From home,
  the chat or anywhere general it is everything. Research says to default to
  everything, because people miss a scope and conclude a thing is not there
  ([NN/g, scoped search](https://www.nngroup.com/articles/scoped-search/),
  [Baymard, search scope](https://baymard.com/blog/search-scope)). So a
  narrowed search is only ever one the person started from that kind's
  place, and it does what those sources ask of one: the scope is said at
  the box and on the results, how many were found elsewhere is on the page,
  widening is one link and first, and when the kind has none, the empty
  state says how many things elsewhere match and links to them.

Not done, and why: suggestions as a person types (a combobox for small
personal data; a lookup already finds records by name); guessing at
misspellings (too many wrong matches on a little data); a scope picker by
the box, a select of kinds (people miss it and forget it is set, NN/g and
Baymard above; the kinds are offered on the results, counted); choosing
several kinds at once as checkboxes (one kind or all covers a person's own
things, and a form of checkboxes needs a button to apply it); kinds with
nothing found shown greyed out (a disabled link cannot be reached or
explained, and zero tells a person nothing they can act on); showing
everything with the place's kind first (still makes a person scan past
what they came from; narrowing with the way out first says the same thing
in fewer steps).
