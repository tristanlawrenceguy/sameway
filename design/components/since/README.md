# since

Back after half an hour or more, a person is shown what other people
changed while they were away, each change with its Undo, at the top of the
pages they open. It stays until they press Got it.

## What it says

First, in one sentence, how many changes and since when: "12 changes by
other people since 2 Jan 14:05. Here until you press Got it." Then the
newest five, each saying who did what and when. The rest are counted in a
link, "7 more changes in Activity", which opens the log where they begin.

## Only others

Their own changes are not news to them, so they are left out; so is what
was said to the assistant, which stays in each person's own chat.

## Kept by each computer

When someone was last about is noted by the computer they use, and never
shared; the changes themselves come from the log, which is.

## Heard where it sits

It is a region under its own heading, first under the page's heading. It
is on the page when the page loads, so it is not announced and does not
take focus; it is met first by anyone reading from the top, and found by
heading or landmark.

## Why it works this way

Working with others means coming back to a place that moved while you were
away. Being told what moved, with the way to take any of it back, is the
difference between trusting a shared place and checking it.

- **How many and since when, first.** Chat apps open the same way: "12 new
  messages since 14:05". The count says how much there is before any of it
  is read; the time says what "since" means
  ([Discord unread banner](https://support.discord.com/hc/en-us/community/posts/19874295797911-Marking-channel-read-iOS),
  [COGA: help users understand what things are](https://www.w3.org/TR/coga-usable/#objective-1-help-users-understand-what-things-are-and-how-to-use-them)).
- **Five at most, the rest a link.** A long absence would otherwise put a
  wall of changes above every page. The link says how many more and opens
  the log where they begin, so nothing is lost
  ([COGA: avoid too much content](https://www.w3.org/TR/coga-usable/#avoid-too-much-content-pattern),
  [GitHub notifications inbox](https://docs.github.com/en/subscriptions-and-notifications/how-tos/viewing-and-triaging-notifications/managing-notifications-from-your-inbox)).
- **It stays until Got it**, and the sentence says so. Nothing closes by
  itself, so a person who looked away or reads slowly does not miss it;
  it is on the page, not over it, so it never interrupts
  ([NN/g: notifications that need acknowledging persist](https://www.nngroup.com/articles/indicators-validations-notifications/),
  [WCAG 2.2.4](https://www.w3.org/WAI/WCAG22/Understanding/interruptions.html)).
- **Got it returns to the same page.** It went Home from most pages before.
- **A region with a heading, not an alert.** Content on the page at load
  is not announced whatever its role, and a region is found by landmark
  ([APG landmark regions](https://www.w3.org/WAI/ARIA/apg/practices/landmark-regions/),
  [GOV.UK notification banner](https://design-system.service.gov.uk/components/notification-banner/)).
- **One anchor per change.** The list leaves the log's anchor to the log,
  so a page showing both has each id once.

Not done, and why: marking it seen just by opening a page, as Slack does
for a channel (a person glancing at the wrong page would lose it);
grouping changes by person or by thing (five lines do not need it, and the
log groups by day); a count in the navigation (a number on every page
invites checking it); announcing it or moving focus to it (it is there on
load, found first where it sits).
