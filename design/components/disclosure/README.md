# disclosure

Use a disclosure for detail that most people, most of the time, do not need
to see: an activity log, raw props, a long aside. It is closed by default and
the summary says what is inside and how many items, so nobody has to open it
to find out.

Native `<details>`, so the state is exposed to every screen reader without
ARIA and the keyboard works everywhere. The enhancement script remembers
whether you opened it, because each action reloads the page.

The content of a closed disclosure is not in the accessibility tree. That is
what makes it quiet, and it is why anything an agent or a screen reader user
must always be able to reach also lives on its own page. The activity log,
for example, is at `/activity` and `/api/activity`.

`count` is how many are inside, and `of` what they are, said after it to a
screen reader: "Activity, 8 changes". A total kept elsewhere belongs on the
link to the whole of it, not on the summary.

Open or closed is remembered for the page, or with `remember: site` for
every page the disclosure is on, as a canvas's side pane is: closed once,
it stays closed. A reply of the assistant, which swaps the activity log in
anew, keeps it as it was. The body moves in only when a person opens it,
not when a page arrives with it open. On paper, whatever is folded away is
printed too.

The twisty is the one sign, across the design system, that a summary opens
something in place: the folds in a chart and a collection draw the same one
(`design/base/27-twisty.css`). It is drawn with borders, so a screen reader
reads nothing for it.

## Why it works this way

- **Native details and summary**, so its state and keyboard need no ARIA
  ([APG disclosure](https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/),
  [GOV.UK details](https://design-system.service.gov.uk/components/details/),
  [Scott O'Hara on details and summary](https://www.scottohara.me/blog/2022/09/12/details-summary.html)).
- **Stays as it was left.** Every action reloads the page, so a fold that
  forgot would snap shut or open again behind the person's back
  ([WCAG 3.2.5](https://www.w3.org/WAI/WCAG22/Understanding/change-on-request.html),
  [WCAG 3.2.3](https://www.w3.org/WAI/WCAG22/Understanding/consistent-navigation.html)).
- **Moves only when opened.** A body that slid in on every page load moved
  whole side panes each time, which is tiring and for some people makes
  them ill ([WCAG 2.3.3](https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html)).
- **One twisty**, drawn with borders, so every fold looks the same and a
  screen reader reads no triangle character before the words
  ([NN/g accordion icons](https://www.nngroup.com/articles/accordion-icons/)).
- **A count of what is inside**, said with what it counts, so the summary
  is true without opening it.
- **Prints open**, so paper holds what the screen folds away
  ([MDN beforeprint](https://developer.mozilla.org/en-US/docs/Web/API/Window/beforeprint_event)).

Not done, and why: a built accordion with buttons and ARIA (details does
it natively, with less to go wrong); a plus and minus sign (a chevron is
the one sign used everywhere here).
