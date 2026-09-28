# Style: air and one blue

What the product looks like, in words, so the assistant and every other
reader build the same thing. The reference is the family of tools people
say they enjoy using: Things, Notion, Linear, Bear, Craft. What they share
is below; nothing here is a taste of our own.

**White, and air.** The page is white. Space does the work borders used to
do: a listing is rows with room around them, groups are separated by a
bold header and one hairline, and a page head has real room under it.
Nothing is boxed unless it is one thing: a record on its own page, a block
on the canvas, a message. Those float on one soft shadow.

**One typeface.** The platform's sans, at reading size (16px, line 1.5).
No serif, no second face. Titles are the same face, bold, tighter
(-0.02em), one or two steps up. Group headers are bold at body size.
Secondary text is the same ink at a lighter strength, never a new colour.

**One ink.** Text is near-black (#1d1d1f), secondary text a grey that still
reads at 7:1 (#45474d). Hairlines are a light cool grey (#e3e3e8); control
edges a mid grey that reaches 3:1 (#85868c).

**One blue.** The accent is one blue (#1a45a8), used for exactly these: the
brand mark, links, the checked box, the primary button, the focus of a
selection, and what the person says in the conversation (on its pale
tint). The assistant's teal marks what it says and made. No third colour
decorates anything.

**Colour says something.** Beyond the two, colour is a state, in words as
well: green is done, blue is a state a thing is in, amber is a day that has
passed or a notice from the software, red is an error. A chip is small and
pale with darker text of the same hue.

**The dot.** A list's colour is a dot, and the same dot goes wherever the
list does: before its name in the sidebar, before the title of its page,
before the way back on one of its records' pages, before a search result
from it, before the caption of a block drawn from it, and at the front of
every row that has no box of its own. Who did a thing is
a dot too, in the actor's colour with a soft halo, before their name on a
message and an activity entry. Today on a calendar is its number in a
filled circle of the blue. A dot is never the only telling: the name,
the word, the date are there beside it.

**A sidebar of lists.** On a wide screen the workspace's lists sit on the
left on the quiet grey, each with a dot in its own colour, the current one
on a darker tint; the rest of the workspace (chat, activity, the design
system) is below them, smaller. The page is the rest of the width, its
content on a reading measure with air on every side.

**The day, the short way.** At the right of a row a day is "Today",
"Tomorrow", "Saturday", "29 Sep", with the time when there is one, in the
quiet ink; today and a day that has passed are amber, with the word. Under
a title the same day is a blue chip in full. What a thing belongs to is
grey words beside its title in a row and a grey chip under a title.

**Rows and checkboxes.** A record with a yes-or-no field shows it as a
checkbox at the front of its row, ticked in the blue, the title beside it
as the link, the day it is due at the right in the quiet ink. A listing of
dated things is grouped by when: overdue, today, this week, later, no date,
done. Done rows are struck through in the quiet ink.

**Small radii.** Controls and rows 8px, boxes 12px, checkboxes 4px, chips
round. Nothing is a sharp rectangle except a table.

**Motion explains.** See [motion.md](motion.md). Nothing moves to look
alive.

## Targets

Nothing a person presses is a small target, and a link is something a
person presses. Three cases, and the link component knows them:

- In a sentence a link stays text, and its hit area reaches half a line
  above and below without moving anything.
- In a row or a chip the whole row or chip is the target: the link fills
  its container (`sw-link--fill`), and anything else pressable in the
  container sits above its reach.
- On its own a link takes the button look (`look: button`): the shape
  and 44px size of a quiet button. The way back to a page, the list
  behind a block, the months either side of a calendar are all this.

Underlined text alone is never the whole target.
