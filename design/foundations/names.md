# Names: what a record is called, what its fields are called, and how two of one name are told apart

Everything a person meets is named: a record by its title, a field by its
label, a control by what it does to what. A name is what a person scans for,
what a screen reader says, and what an agent finds a control by, so it is
worked out in one place for each and said the same on every surface: a
list, a record's page, search, the log, the canvas, the API.

## A record's name

A record is called by its title field, as the schema names it, or else by
the first thing it says (`schema.Called`, `records.Name`). It is said whole
where there is room (a page's heading, a search result) and cut at a word,
with an ellipsis, only where there is one line (a row, the window's title,
the crumbs), the whole kept for a pointer that rests on it.

## A field's label

A field is called by its label in the schema, else by its name made
readable (`schema.Field.Display`): Due, Type, URL, HTTP method, How often.
A record's page lists its fields as label and value, the way a summary list
does ([GOV.UK, summary list](https://design-system.service.gov.uk/components/summary-list/)):
the label says what the value is, so it is there on purpose, and the label
a person would say is the right one. A value is said as a person reads it:
a choice by its label (At least the target, not reach), a day as glance.md
says, yes or no for a yes-or-no.

A field nobody chose (its default, or no) is not listed
(`schema.Field.Unchosen`); the page says what was chosen about the record.

## Two records of one name

Two records with one name make two controls with one name: two boxes
"Done Buy coffee", two links "Buy coffee" to different pages. A person
moving by links or controls cannot tell them apart, and neither can an
agent finding one by role and name
([WCAG 2.4.6](https://www.w3.org/WAI/WCAG22/Understanding/headings-and-labels.html),
[WCAG 2.4.9](https://www.w3.org/WAI/WCAG22/Understanding/link-purpose-link-only.html)).
So where records of one name are shown together, each is told apart by the
first thing about them that differs that a person would say, the same way on
every surface (`blocks.Apart`, `RecordWays`):

1. **Its kind**, where kinds are mixed (search): Buy groceries — Note, Buy
   groceries — Task.
2. **Its day**, as its glance says it: Buy groceries (due Fri 16 Oct).
3. **When it was added**, to the minute, then to the second: (added Sat
   26 Sep at 12:53pm).
4. Only when nothing else differs, **where it comes**: the first of 2,
   counted in the order they were added, so a record is the same number in
   its list and in the log.

The words are read after the name, in the link and in the box alike, and
only where the name repeats: a name that is its own is said alone. "Due"
and "added" are words a person uses ("the one due Friday"), not stored
names.

A name that repeats is often one thing twice. Those nothing tells apart
are offered on their list as one to keep, and the assistant is told when
it makes one called as an open one is (records.Alike, records.Twins,
keep_one). Telling them apart is for what is really two.

## Decided, so not findings

- A label beside a value on a record's page (Type: Timer, URL, Due).
- The words that tell apart records of one name, said as above, where the
  name repeats; and the kind of a search result where kinds are mixed.
- "first of 2" where nothing else differs. If records called the same are
  one thing twice, the finding is that they are there twice, not how they
  are told apart.

A page breaks these rules when it shows a stored name (`created_at`,
`test_type`, `kind` in lower case), a value as stored where it has a label,
two controls with one name, or the words that tell apart where the name is
its own.
