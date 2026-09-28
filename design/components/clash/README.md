# clash

Two people changed the same text at once, each on their own computer,
neither seeing the other's change. Every copy of the workspace keeps the
later version, so they agree; this offers the other one back, on the
record's page, until someone chooses.

## What it shows

The other version whole, as it was written, so it reads and copies as it
is. Under it, closed until opened, what differs from the page's version,
word by word: the page's words struck through, this one's underlined. A
line above says which is which, and a screen reader hears "only in the
page's version" or "only in this one" around each change, once.

## Three answers

- **Use this version** puts it in place of the page's.
- **Keep both** puts the page's version first and this one after a blank
  line, to be joined by editing. Nothing is lost.
- **Keep the page's version** sets this one aside.

Each answer is said on the page afterwards, with its Undo. Undoing Keep the
page's version brings the offer back.

## Not every edit

Writing after reading the other person's change is an ordinary edit, not a
clash. Only two versions each written without the other are.

## Why it works this way

- **Keep the other version and say so.** Latest-wins keeps every copy the
  same, but alone it throws someone's words away without telling them, as
  Notion's offline edits can. Dropbox keeps a "conflicted copy" for the
  same reason ([Dropbox](https://help.dropbox.com/organize/conflicted-copy),
  [Notion offline conflicts](https://backups.so/blog/notion-offline-sync-conflicts-data-loss)).
- **Keep both is an answer.** Git tools offer Accept Both and iCloud lets
  you keep both documents, because often each version has something the
  other lacks ([VS Code](https://code.visualstudio.com/docs/sourcecontrol/merge-conflicts),
  [Apple](https://support.apple.com/en-gb/guide/pages-icloud/gil07d27350d/icloud)).
- **Show what differs.** Two long texts that differ by one sentence are
  hard to compare by eye. Wikipedia's edit conflict page and VS Code's
  Compare Changes mark the changed words
  ([MediaWiki](https://www.mediawiki.org/wiki/Help:Two_Column_Edit_Conflict_View)).
- **Marked by shape and words, not colour.** Struck through and underlined
  survive forced colours, and a line in view says what each means
  ([WCAG 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)).
- **Heard as words, once.** Screen readers differ on `del` and `ins`: NVDA
  says them, JAWS added it and took it out again. Plain spans with hidden
  words say the same thing everywhere, and only once
  ([Adrian Roselli](https://adrianroselli.com/2017/12/tweaking-text-level-styles.html)).
- **Named, not placed.** The buttons say "the page's version", not "the
  one below", which means nothing to someone who cannot see the layout
  ([WCAG 1.3.3](https://www.w3.org/WAI/WCAG22/Understanding/sensory-characteristics.html)).
- **Every answer can be undone**, and the outcome is said where the person
  is, back on the record's page. Before, a choice sent them to the home
  page with no word of what happened, and Use could not be undone
  ([NN/g, user control and freedom](https://www.nngroup.com/articles/user-control-and-freedom/)).
- **The whole text stays in view.** Wikipedia's conflict page offers to
  copy your full text first; here it is shown whole, not only as a diff.

Not done, and why: choosing paragraph by paragraph, as Wikipedia does (much
more to learn for a rare event; Keep both and then editing does the same);
showing the two side by side (it does not fit at 320px, and the page's
version is already on the page); colour tints on the changes (shape and
words already say it, and tints vanish in forced colours); undoing Use
this version or Keep both bringing the offer back (the undo puts the text
back as it was; the other version is still in the log's entry).
