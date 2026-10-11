# Save to Sameway

Keeping a page, a photo or a few words the moment they are seen: from the
Save to Sameway bookmark on a computer, from a phone's Share menu (the
installed app is a share target), or from the form on /share. What
arrives is one note. The code is `internal/server/share*.go`; the tests
are `share_test.go` and `share_guard_test.go`.

## Why it works this way

- **One press, and the page stays.** The bookmark opens /share in a new
  tab, filled in, with the form first: it is all the person came for. The
  page they were reading is not left. Saving goes to the new note with one
  message, "Saved", and Undo, said once by the outcome; nothing else is
  live. A phone's share is answered with a redirect (303), so a reload
  never saves twice ([MDN, share_target](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Manifest/Reference/share_target)).
- **What is kept.** The page's own title (og:title, else its title), the
  link, and its main words as Markdown, read from the article, else the
  main part, else the body. Taken out: scripts, menus, headers, footers,
  forms, dialogs, and whatever the page itself marks as around its words
  (the landmark roles navigation, banner, contentinfo, complementary and
  search) or hidden (`hidden`, `aria-hidden="true"`). A page in another
  encoding than UTF-8 is read in the one it names. The words can then be
  found by search and read if the page goes, which is why read-later apps
  keep them (Pocket, Instapaper).
- **A link without its tracking.** The click identifiers Safari's Link
  Tracking Protection removes (gclid, fbclid, msclkid, dclid, twclid,
  igshid and the like) and the campaign tags (utm_) are taken off. Opened
  from a note later, the page is not told it came from that post, and the
  same page from two places is one link.
- **Saved once.** The same link shared again with nothing new, no file and
  no words the note does not have, opens the note that keeps it and says
  "Already saved" with the day. Pocket keeps one copy when save is pressed
  again; its list fills with duplicates when the same page arrives with
  different tracking ([Oswalt](https://oswalt.dev/2015/01/remove-duplicates-from-pocket-list/)),
  which the line above prevents. New words with the
  link are a note of their own: the person said something new.
- **Never a page nearby.** Sameway reads a shared link from the person's
  own computer, so a link, or a page it redirects to, must not make it
  read what only that computer or its network should see (a router, a
  printer, another program, a cloud machine's metadata). Each address is
  checked as it is dialled, after the name is looked up, so redirects and
  names that change are checked too: loopback, private (RFC 1918 and
  fc00::/7), link-local, carrier and Tailscale (100.64.0.0/10), multicast,
  and 0.0.0.0/8 are refused ([OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)).
  No proxy is used, since a proxy would read past the check. The link is
  still kept; the message says why its page was not read.
- **Limits.** 5 MB of a page, 40,000 characters of its words, and ten
  seconds: past that attention goes ([NN/g, response times](https://www.nngroup.com/articles/response-times-3-important-limits/)),
  and a slow page is kept as its link. Only http and https are read.
- **A problem keeps what came.** A link that cannot be kept, or nothing to
  save, returns to the form filled again with the title, link and words,
  and the error summary points at the Link field, as every form here does.
  A file has to be chosen again: a page cannot be given one back.
- **Every way in.** The bookmark is a link to drag to the bookmarks bar,
  and its address is also in a field to copy, for a keyboard, and for an
  iPhone or iPad, where a bookmark is made and then edited
  ([Instapaper's way](https://morrick.me/archives/2703)). Dragging is the
  browser's, not the page's, so WCAG 2.5.7 does not ask for it
  ([Understanding 2.5.7](https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html)),
  but a person without a mouse still needs a way. An iPhone does not let a
  web app into its Share menu ([WebKit bug 194593](https://bugs.webkit.org/show_bug.cgi?id=194593)),
  so the page says to use the bookmark there; Android, and Chrome or Edge
  on a computer, list the installed app.
- **Agents.** `POST /share` is a page action an agent can send too, with
  the same words back. It reaches outside the workspace, so a dry run does
  not try it.

## Not done, and why

- **Saving the moment the bookmark is pressed.** Any site could open
  /share with a link of its choosing, so the bookmark only fills the form,
  and the person presses Save. Another site's page cannot post to /share either:
  the server refuses a browser's write that is not from Sameway's own
  pages (ARCHITECTURE.md, "Other sites").
- **Reading the page later, in the background.** Pocket saves at once and
  reads after. Here the person waits up to ten seconds, which keeps one
  note, one message and one Undo; a slow page is kept as its link.
- **A full Readability port** (scoring every block for its text). The
  article, main and landmark rules cover most pages without it.
- **Reading through a proxy.** It would be read past the check above; a
  person whose network needs one keeps the link.
- **Following the page's canonical link.** The address the person shared
  is the one they know; tracking is taken off instead.
