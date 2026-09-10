# Typography

System font stack, so nothing loads from a network and every platform's
best text rendering is used. `font-feature-settings` turns on tabular
numerals (`tnum`) so times, counts, and table columns line up.

| Token | Size | Use |
|---|---|---|
| `text-2xl` | fluid 2rem to 2.75rem | Page title (h1) |
| `text-xl` | fluid 1.5rem to 1.9rem | Region headings (h2) |
| `text-lg` | fluid 1.2rem to 1.35rem | Card titles, brand |
| `text-md` | 1.0625rem | Body. Slightly above 16px on purpose. |
| `text-sm` | 0.9375rem | Hints, meta, status |
| `text-xs` | 0.8125rem | Badges, timestamps. Never for body text. |

Line height is 1.6 for reading and 1.2 for headings. Headings use
`text-wrap: balance` and slight negative tracking. The reading measure is
capped at 68 characters (`--sw-size-measure`) on prose, cards, and messages.

No italics for emphasis in components; weight and colour do that work, and
the actor's name is always a bold word before the colour is seen.
