# search

One search over everything a person has. A form with a labelled field and
a button, marked as a search landmark, that sends its words to `/search`
as a GET, so the results page is a link that can be kept and shared. The
same search is behind `/api/search`, the assistant's search tool and
`sameway search`.

It belongs in the header, where a person reaches for it on every page; at
compact size it takes less room, and at icon size it is a glyph with its
name that opens the search page. In a bar the visible label is the button
text, and the field keeps its accessible name.

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

Not done, and why: suggestions as a person types (a combobox for small
personal data; a lookup already finds records by name); guessing at
misspellings (too many wrong matches on a little data).
