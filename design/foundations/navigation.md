# Navigation: the sidebar and the way around

The sidebar is the workspace's lists, each with its dot, then the rest of
the workspace (Search, Chat, Activity, Workspaces, Help) quieter below.
On a wide screen it stands on the left, always in view; on a narrow one
the same links run in one row along the top and scroll away with the
page. It is built in `internal/server/nav_nest.go` and
`internal/render/layout.html`, styled in `design/base/06-shell.css`.

## Why it works this way

- **Links in lists, not a tree.** Each list is a link; a list the person
  asked to keep in reach (`ui.nest`) has its records as a list of links
  inside its item. A screen reader hears "Main, navigation", then each
  list with how many items it holds, and moves with Tab like any link.
  The ARIA tree role is for working on a hierarchy (a file manager), and
  needs arrow keys, roving focus and expand state a person must learn;
  as site navigation it tested badly with touch and screen reader users
  ([Adrian Roselli, Be careful using menu](https://adrianroselli.com/2023/05/be-careful-using-menu.html),
  [APG disclosure navigation](https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/examples/disclosure-navigation/),
  which leaves out the menu role for the same reason).
- **Where you are, said the right way.** The link to the page open now
  carries `aria-current="page"`. On a page inside a list (one of its
  records, a new one) the list's link carries `aria-current="true"`: the
  person is in it, but it is not the page, so a screen reader does not
  call it the current page. GOV.UK's service navigation does the same,
  `page` for the page and `true` for the section it is in
  ([GOV.UK service navigation](https://design-system.service.gov.uk/components/service-navigation/)).
  Both look the same, with a bar at the edge and weight as well as
  colour ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
  When a nested record is open it holds the mark and its list does not,
  so it is said once.
- **Nested only when asked.** A record under its list is one click away,
  but every one added makes the sidebar longer to scan for everybody.
  So nothing is nested until the person wants it (the assistant sets
  `ui.nest +habit` from what they say). Notion leaves a database's pages
  out of the sidebar for the same reason: there can be thousands
  ([Notion](https://twitter.com/NotionHQ/status/1233181924467732481)).
- **A cut list says it is cut.** Twenty records at most, newest first,
  archived ones left out as the list leaves them; past that a
  last link, "See all 34 habits", says how many there are and leads to
  them. Notion's sidebar ends a section shown in part with a More link
  ([Notion, the sidebar](https://www.notion.com/help/navigate-with-the-sidebar)).
- **The lists that matter, in a stable order.** A list shows once it has
  something in it or the person made it (`ui.lists: all` shows every
  one), in the workspace's order, so the same item is in the same place
  on every page ([WCAG 3.2.3](https://www.w3.org/WAI/WCAG22/Understanding/consistent-navigation.html)).
  Left, labelled in words and always in view, the way vertical
  navigation is easiest to scan ([NN/g, vertical navigation](https://www.nngroup.com/articles/vertical-nav/)).
- **Named landmarks.** "Main" for the lists and "This workspace" for the
  rest, so the two navigations are told apart in a landmark list without
  repeating the word navigation, which the role already says.

## Not done, and why

- **No folding.** A list's records cannot be folded away by a twisty,
  and nothing remembers what was open. Asking is the switch: `ui.nest`
  is the remembered state, for this person, on every device. A twisty
  on every list would be one more control on every line, and a
  remembered fold is state a person forgets they set.
- **Nested records only on a wide screen.** The row along the top of a
  phone has no room; the list's page names every record, and is one tap
  away. A menu button that hides the lists behind it would cost every
  visit a tap, where the row costs a swipe.
- **No drag to reorder, no favourites.** Linear and Notion let a person
  arrange the sidebar by hand. Here the assistant arranges on request,
  and a list's order is the same everywhere it is shown.
- **look does not report aria-current.** An agent reads `/api/look` for
  the address it asked for, so it knows where it is; the state is for a
  person who arrived by a link.
