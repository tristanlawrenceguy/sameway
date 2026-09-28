# presence

Who else is in the workspace just now, and where: "Also here: Hana, on
Shopping list". It is there only while someone else is, so most of the
time the page says nothing about it at all.

## Who counts

Someone with a page open who has touched it in the last ten minutes, or
who moved about in the last half minute, on this computer or on another
that hosts the same workspace. A page left open with nobody at it stops
counting: after ten minutes with no key, pointer or wheel it tells the
server its person is idle, and still follows changes. A tab in the
background does not count at all. Never the one reading: nobody needs
telling that they are here.

## Where

Where someone is, by the name of the page: a list, a record, Home. On the
page the reader is on, it is "on this page", which is when two people
might set about the same thing. Where the owner is on a part that is
theirs alone, such as one of their conversations, only that they are here
is said, since its title is not the others' to read. A published page
never carries the line, not even hidden.

## Heard, not announced

It is a sentence, read where it sits. The line is not a live region:
people come and go, and a screen reader announcing each would interrupt
whatever its user is doing. It is found by reading the header, like
anything else there.

One thing is said: someone arriving on the page the reader is on, as
"Hana is on this page too", politely, once per person while the page is
open, however often they come and go. Leaving is never said; the line
shows it.

## Why it works this way

- **Quiet, found by reading.** Two people who can see each other in the
  same place do not set about the same thing at once. The line is quiet so
  that it is noticed when it matters and not otherwise. A page that says
  too much is "chatty" to a screen reader, and status messages are not
  meant to be made up for everything
  ([WCAG 4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html)).
- **One arrival is said: onto this page.** That is the one that changes
  what the reader does next. Google Docs says who enters and who is
  editing near the reader, and lets them turn it off; saying only arrivals
  on the same page, once each, keeps the useful part without the noise
  ([Google Docs with a screen reader](https://support.google.com/docs/answer/6239410?hl=en)).
- **Said through a status that is there from the start.** A live region
  put on the page with its words already in is not reliably read, so the
  empty status is on the page when it loads, outside the header the page
  replaces as it follows
  ([Scott O'Hara, Are we live?](https://www.scottohara.me/blog/2022/02/05/are-we-live.html)).
- **Idle is away.** Someone who left their page open an hour ago is not
  here; the list would say they are and hold the others back. Google
  Docs lists only active collaborators, and Teams shows a person away
  after five minutes without input; ten keeps a slow reader counted
  ([Google Docs](https://support.google.com/docs/answer/6239410?hl=en),
  [Microsoft Q&A](https://learn.microsoft.com/en-us/answers/questions/4396080/how-to-stop-teams-from-showing-away-in-5-minutes)).
- **Only what the reader may see.** Where someone is is said only where
  the reader could go too, and the internet is told nothing: being seen
  is shared with the people in the workspace, not further.
- **Names carry it.** The dot is decoration hidden from screen readers;
  in forced colours it is drawn in the text colour. Nothing moves, so
  there is nothing to reduce for reduced motion (see person).

Not done, and why: announcing every arrival and departure (heard all day,
it is noise, and the line already says it); announcing departures at all
(nothing to do about them); an "away" state shown beside names (someone
idle is simply not here, which is plainer); an invisible mode to hide
oneself (everyone here was let in by the owner to work together, and a
hidden person editing beside you is the clash the line exists to avoid);
counting a screen reader user's reading as activity (browse mode keeps
its keys to itself, so ten minutes of reading without a key reaching the
page can count as idle; the cost is only that they drop off the others'
line until they next press a key); a setting to turn the arrival off (it
is one polite sentence, at most once per person).
