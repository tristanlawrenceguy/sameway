# Text: how words are found, named, put in order and shown

People write in every language, on phones that curl their quotes and on
Macs that store é as e and an accent. Sameway compares words in one place,
`internal/fold`, so search, a condition, a list's order and two names that
look the same agree on every surface: a page, the API, MCP, the CLI and
what the assistant reads.

## Three rules for three jobs

| Job | Rule | Café | Straße | Mum’s | Item 2, Item 10 |
| --- | --- | --- | --- | --- | --- |
| **Finding** (search, `where title~…`) | `fold.String`: case, accents and width off, letters spelled plainly | cafe finds it | strasse finds it | mum's finds it | |
| **One name** (`title=…`, records alike, names told apart, tags, a person or canvas by name) | `fold.Key`: same letters in any case or stored form | Cafe is another name | STRASSE is it | Mum's is it | |
| **Order** (a list by title, choices, tags) | `fold.Compare`: Unicode collation, numbers by value | with the E's | | | 2 before 10 |

Finding forgives most, because a person without the letter on their
keyboard types it plainly: Łódź as lodz, København as kobenhavn,
Diyarbakır as diyarbakir, a full-width ＣＡＦＥ as cafe. One name forgives
only what nobody can see or hear: é stored two ways, case, a curly
apostrophe, two spaces. Accents still make two names, since Cafe and Café
are spelled differently.

## Shown as written

A title is kept as it was written, and shown so: cut only at the edge of
what a person sees as one character (#691), and a link by a title with
brackets is still a link (#693). Words in Hebrew or Arabic read right to
left: a page's heading, a record's title, a link, a text field and a text
box take their direction from their own first letter (`dir="auto"`), so
the ? at the end of a right-to-left title stays at its end and does not
pull the words beside it out of order. A long word with no spaces breaks
rather than run off a 320px screen.

## Why it works this way

- **One name is a canonical caseless match.** The W3C says case-insensitive
  matching of names should normalise, fold case in full, then normalise
  again, and should not fold compatibility forms, which take meaning away
  ([W3C Character Model: String Matching](https://www.w3.org/TR/charmod-norm/),
  [UAX #15](https://unicode.org/reports/tr15/)). So two titles that look
  identical, one stored as NFC and one as NFD, were two names: never alike,
  and two links a screen reader said alike were not told apart. Now they
  are one. Full case folding makes STRASSE straße and ΣΟΦΟΣ σοφος, which
  lowercasing missed.
- **Finding is a folding, as search engines do it.** ICU folding, from
  Unicode's character foldings (UTR #30), takes off case, accents and width
  together, and an ASCII folding spells ł, ø, ß and æ plainly
  ([Elasticsearch ICU folding](https://www.elastic.co/guide/en/elasticsearch/plugins/current/analysis-icu-folding.html)).
  Search and conditions share the one function, so they cannot drift.
- **Order is the Unicode Collation Algorithm.** Byte order put Émile after
  Zoë and Item 10 before Item 2. The UCA's default order (CLDR root) sorts
  accents and case as small differences after the letters, and CLDR's
  numeric option sorts digits by value
  ([UTS #10](https://unicode.org/reports/tr10/),
  [UTS #35 collation options](https://unicode.org/reports/tr35/tr35-collation.html#Setting_Options),
  [Go collate](https://pkg.go.dev/golang.org/x/text/collate)).
- **Direction from the text itself.** For text whose direction is not known,
  such as what a person types, the W3C says to put `dir="auto"` on the
  element that holds it, which isolates it and takes its direction from its
  first strong letter
  ([W3C inline bidi markup](https://www.w3.org/International/articles/inline-bidi-markup/),
  [HTML dir](https://html.spec.whatwg.org/multipage/dom.html#the-dir-attribute)).
  The page heading does it in CSS (`unicode-bidi: plaintext`), the same
  rule. A person's name already sits in a `bdi` (person component).

Not done, and why:

- **No language per title.** WCAG 3.1.2 exempts proper names and words of
  indeterminate language, and Sameway cannot know what language a person
  wrote a title in; guessing wrong would make a screen reader read it in
  the wrong voice
  ([WCAG 3.1.2](https://www.w3.org/WAI/WCAG22/Understanding/language-of-parts.html)).
- **No order for one language.** Swedish puts å after z and German phone
  books put ä as ae; the root order suits every language passably and none
  wrongly, and a workspace does not say its language.
- **Compatibility forms are not one name.** ＣＡＦＥ and CAFE are found by
  one search but stay two names, as the W3C advises.
- **Marks are dropped in finding for every script.** Taking off Mn marks
  also takes off Japanese voicing marks and Hebrew points, which is what a
  UCA primary-strength match does too; finding stays forgiving.
- **Turkish case is not tailored.** Without a language, İ and I fold to i
  for finding, as most keyboards type them.
