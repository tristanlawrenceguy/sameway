# Typography

One typeface: the platform's sans, so nothing loads from a network and
every platform's best text rendering is used. Titles are the same face,
bold and a little tighter; hierarchy comes from size, weight and the
strength of the ink, never from a second face. `font-feature-settings`
turns on tabular numerals (`tnum`) so times, counts, and table columns
line up.

| Token | Size | Use |
|---|---|---|
| `text-2xl` | fluid 1.75rem to 2rem | Page title (h1) |
| `text-xl` | 1.375rem | Region headings (h2) |
| `text-lg` | 1.125rem | Group headers, card titles, brand |
| `text-md` | 1rem | Body, and the title of a row |
| `text-sm` | 0.875rem | Hints, meta, the day at the right of a row |
| `text-xs` | 0.8125rem | Chips, timestamps. Never for body text. |

Line height is 1.5 for reading and 1.2 for headings. Headings use
`text-wrap: balance` and slight negative tracking. The reading measure is
capped at 68 characters (`--sw-size-measure`) on prose, cards, and messages.

No italics for emphasis in components; weight and colour do that work, and
the actor's name is always a bold word before the colour is seen.
