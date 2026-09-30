# filters

One way to narrow a list: a set of results, a collection, the activity
log, a calendar. It comes in two shapes, and which one is decided by what
is being chosen, not by the page:

- **links**, when there is **one choice among a few**, about seven or
  fewer, such as the kind of a search result. Each answer is a link to the
  same list narrowed, with how many it holds: All (5), Tasks (3), Notes
  (2). Pressing one shows it; the one shown is marked `aria-current="page"`.
  All, or whatever stands for no narrowing, comes first.
- **form**, when there are **several choices made together**, such as who
  made a change, what it was to and when. Each is a labelled native
  select; nothing changes until Apply. Under the form the list says how
  many match and, once, what it shows, in words, with Reset beside it:
  "1 task. Showing: not done, due before today. Reset".

Either way the choices are in the page's address, so it works with no
script, a reload or a link keeps them, and Back goes back. The component
draws; the page decides. Give it the address to come back to (`action`,
with `#the-list` so the page lands on it), the page's other address fields
to keep (`keep`, sent on as hidden inputs), the choices (each a `name`,
a `label` and `options` of `value`, `label`, `selected`, and for links an
`href` and a `count`), what is shown in words (`count`, `showing`) and the
way back (`reset`), and for those who may change the list Keep these
choices (`save`: the address to POST to and its hidden fields), which
makes the choices shown the list's own setup. The server decides which choices make sense and
applies them, and only adds to what a list was set up to show, never takes
from it.

Put it just over what it narrows. When two lists on one page each have
their own, name each field after its list (`c-<block>-sort`) and give
`context`, the list's name, which Apply and Reset carry unseen: "Apply to
Tasks with a day". The legend can be for screen readers only
(`hideLabel`) where a heading just above says what the list is; otherwise
it is in sight.

## What a screen reader hears

links: "Kinds of result, navigation, list, 3 items, All (5), current page,
link; Tasks (3), link". Pressing one is a new page, whose heading and
window title say what it shows.

form: "Show, group; Who, Anyone, pop-up button" for each select in turn,
then "Apply, button". After Apply the page comes back at the list, and the
line under the form says how many match and what is shown, and "Reset,
link", then, where offered, "Keep these choices for Tasks, button". Moving through a select's options says each one and moves nothing.

## Why it works this way

- **Links for one choice among a few, a form for several.** A link
  pressed is going somewhere, which a person expects, so a single choice
  can be applied at once and each answer is an address that can be kept
  ([BBC GEL, filter and sort](https://bbc.github.io/gel/components/filter-and-sort/)).
  Several choices set together are a batch: a person who knows what they
  want sets them all, then applies once, rather than wait for a new page
  after each ([NN/g, batch and interactive filters](https://www.nngroup.com/articles/applying-filters/),
  [DWP research on filters](https://design-system.dwp.gov.uk/research/filters),
  [MoJ filter a list](https://design-patterns.service.justice.gov.uk/patterns/filter-a-list/)).
  One kind of thing is a filter, not facets: there is one thing to narrow
  by ([NN/g, filters vs facets](https://www.nngroup.com/articles/filters-vs-facets/)).
- **Apply, never on change.** Choosing an option in a select changes
  nothing until Apply: a new page on every change moves the ground under a
  keyboard and a screen reader, and a person setting two choices waits for
  one ([WCAG 3.2.2](https://www.w3.org/WAI/WCAG22/Understanding/on-input.html),
  [W3C H32](https://www.w3.org/WAI/WCAG22/Techniques/html/H32)).
- **In the address, on the server.** A GET form with a button and plain
  links work with no script; the address says what is shown, so it can be
  kept, sent and reloaded ([GOV.UK finder-frontend](https://github.com/alphagov/finder-frontend)).
  Choices do not follow a person to their next visit: the address keeps
  them, which is enough (MoJ, above).
- **What is shown, said once, with a way back.** The choices in words
  next to the results, not a count of filters, and one Reset
  ([Baymard on applied filters](https://baymard.com/blog/how-to-design-applied-filters),
  DWP above). How many match is said in words.
- **Counts on links, not on a form's options.** A row of links counts
  what one search or one month already holds; counting each option of
  several selects would be a query per option on every page.
- **Narrow, never widen** (DWP above: filters are AND, each one takes
  away). The address can only add what the choices offer; a list set up
  to show undone tasks cannot be made to show done ones by a link.
- **Kept when the person says so.** A look stays a look: the address
  holds it and nothing is saved. Keep these choices, next to Reset and
  only where a person may change the list, makes it the list's setup, a
  plain POST that narrows it the same way and is undone like any edit.
  Without it, choices that read "Showing: not done, due soonest first"
  looked kept and were not (the agent evaluation, T5).
- **Plain words, few of them.** Any or All first, each answer a short
  phrase, at most four selects ([W3C COGA](https://www.w3.org/TR/coga-usable/)).
- **Links look like links; the one shown does not.** The others are
  underlined, pressable without a pointer to find them; the one shown has
  a bar, weight and the text colour, and keeps its bar alone in forced
  colours.

Not done, and why: radios in the form (the research prefers them to
dropdowns, but three groups of them are taller than many lists; each
select here has a few short options, labelled in sight); submitting a
select on change (3.2.2, above); a live region for the new count (the
results are a new page, whose title says it); checkboxes to pick several
kinds at once (one kind or all covers a person's own things, and a form
of checkboxes needs Apply, which makes one quick choice two steps);
Keep these choices on the links shape (one choice among a few is a
place to go, not a setup; a calendar's kinds come with its several
types); a confirmation before keeping (Undo is on the outcome); removable
tags for each applied filter (with three selects at most, one
Reset and the selects themselves, which keep what was chosen, say the
same thing); a panel of filters that slides in on a phone (it hides the
choices, and a few selects wrap onto their own lines instead).
