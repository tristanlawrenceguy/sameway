# Letting people in

A workspace is one person's, on their computer. They can let others open
it: their own phone on the same Wi-Fi, someone in the house or the office
invited by name, someone further away over Tailscale. This is where those
rules are, and why. The code is `internal/server/phone_lan*.go`,
`phone_pair.go`, `lan_invite.go`, `access_list.go` and
`internal/chat/access.go`, `invite.go`.

## What the owner sees, on Workspaces

1. **On your phone.** Off: one button, and before it, on which Wi-Fi to
   use it. On: the code to scan, with its letters to type beside it; the
   form to invite someone by name; the devices paired, each with Remove;
   and Stop answering on the Wi-Fi.
2. **Who can open this.** Everyone let in, however they came, one row
   each: their name, what they may do in words ("may look", "may edit"),
   how they come in ("On Tailscale as bob@example.com", "1 device on the
   Wi-Fi", "An invite not opened yet, until 17 Oct"), and **Take away
   access**. With nobody, it says "Only you" and where to invite someone.

## Why it works this way

- **People, not devices, are what the owner shares with.** Google Drive's
  "People with access" and Notion's share menu list each person with
  their role and a Remove access beside it, where sharing is done
  ([Drive](https://support.google.com/drive/answer/2494893),
  [Notion](https://www.notion.com/help/guides/understanding-notions-sharing-settings)).
  Before this list, someone invited but not yet in was shown nowhere, and
  taking access from someone invited on the Wi-Fi meant removing each of
  their devices while their invite still worked.
- **Taking access away is one press, not asked, and can be undone.**
  Giving access is asked (a yes in the chat for Tailscale, the owner's
  own form for the Wi-Fi); taking it back is never in the way. Every
  device of theirs is shut out at once and an invite not yet opened stops
  working. Undo gives the access back; the invite stays gone. W3C's
  [Making Content Usable](https://www.w3.org/TR/coga-usable/) asks that
  people can go back and undo, and that results of an action are said
  (4.5.2, 4.8.3): the message says "from any device".
- **Roles are said in words, as what the person could do.** "Edit: add
  and change things" and "Look: read only", and "may look" in the list,
  not Viewer and Editor.
- **The code to scan is also eight letters to type.** A QR code must
  never be the only way
  ([Section508.gov](https://www.section508.gov/blog/accessibility-bytes/qr-codes/)):
  not everyone can aim a camera. WhatsApp links a device with an
  eight-character code for the same reason
  ([WhatsApp](https://faq.whatsapp.com/1324084875126592/)). The letters
  follow [RFC 8628](https://datatracker.ietf.org/doc/html/rfc8628#section-6.1):
  twenty consonants, no digits, two groups of four, read however they are
  typed. Five wrong tries drop every code waiting, so eight letters are
  enough (the RFC's own sum). The page a phone not paired sees has the box
  to type them in.
- **An invite is a link, to send by message.** It is copied and pasted,
  never typed, which WCAG's
  [Accessible Authentication](https://www.w3.org/WAI/WCAG22/Understanding/accessible-authentication-minimum.html)
  counts as no test of memory. It works once, within a week.
- **On which Wi-Fi is said before it is turned on.** Sameway answers on
  the Wi-Fi in plain HTTP. On a home or office network with a password
  that is the family's or the team's, that is fine; on a café's or a
  hotel's, anyone on it could read what passes
  ([NCSC](https://www.ncsc.gov.uk/collection/zero-trust/architecture-design-principles/dont-trust-any-network)).
  The page says so where the choice is made.
- **The assistant does the same.** `let_in` gives access (asked) and takes
  it away (`access: none`), by email or, for someone invited on the Wi-Fi,
  by name. Making an invite and pairing a phone stay the owner's, face to
  face: a link that lets someone in should not be made by a sentence in a
  chat.

## Not done, and why

- **Changing someone's role from the list.** Inviting them again by the
  same name changes it, and is rare; a menu on each row is more to read.
- **Expiry dates on access.** Drive offers them for work files. In a home,
  taking it back by hand when it is no longer wanted is simpler.
- **When each device was last seen.** It would help to spot a device not
  recognised, but needs a write on every request from a phone.
- **HTTPS on the Wi-Fi.** A certificate for a private address means
  trusting a home-made authority on every phone, which nobody will do.
  Tailscale is the encrypted way, and is offered.
