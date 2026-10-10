# Email in

A person forwards a booking, a bill or a note to self to their +sameway
address, or moves it to the Sameway folder, and it comes in as an email
to sort on Today. Conversations are worked out from the headers, the
person's own replies are read from Sent, and Today says whose turn it is.
The code is `internal/mailin` (reading the mailbox) and
`internal/server/mail_*.go` (the pages).

## Why it works this way

- **The job is getting it out of the inbox, once.** Todoist, Things and
  OmniFocus each give an address to forward to, and the subject becomes
  the title. Sameway does the same, but keeps the email whole and asks
  on Today what it is: a task, or done with.
  ([Todoist](https://www.todoist.com/help/articles/forward-emails-to-todoist-JPJ1V339),
  [Mail to Things](https://culturedcode.com/things/support/articles/2908262/))
- **One story per mail service.** A person reads only their own
  service's steps. Gmail needs an app password, and 2-Step Verification
  for one; Google turned off plain passwords for IMAP in 2025, so an app
  password or OAuth is the only way in.
  ([Google](https://support.google.com/a/answer/14114704?hl=en))
- **No picture is fetched from the sender.** A picture in an email is
  loaded from the sender's server when the email is opened, and that load
  tells the sender it was read, when and from where. HEY strips these spy
  pixels; Sameway turns each picture from elsewhere into what it shows,
  and leaves out one that shows nothing. Links stay: following one is the
  person's choice. ([HEY](https://www.hey.com/spy-trackers/))
- **An email's words are its sender's.** An agent reading an email is told
  it was written by "an email from" its sender, never by Sameway or an
  action, because anyone can send an email that speaks to an agent
  (OWASP LLM01, indirect prompt injection). The sender's name is what the
  email says, which anyone can fake.
  ([OWASP](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html))
- **Conversations come from the headers.** Message-Id, In-Reply-To and
  References tie a reply to what it answers, as in Thunderbird and most
  mail apps (jwz's threading).
  ([jwz](https://www.jwz.org/doc/threading.html))
- **Whose turn is the latest email's own words.** Who wrote last, and
  whether they asked. Only what they wrote this time counts: lines quoted
  with >, everything under "On ... wrote:", "Original Message" or a
  signature, and a link's address (its ? is not a question) are left out.
  Measured against a small model, the rule was right 11 times in 12
  (`internal/bench` TestThreads).
- **A turn can be put down.** HEY's Reply Later is marked done and
  Superhuman cancels a reminder when the reply comes. Here the person's
  reply from Sent clears it, and **No reply needed** on Today clears it
  when they answered another way or need not. It is update_record of the
  email's turn, so an agent can do the same, and it is undone like any
  change. The next email decides again.
  ([HEY](https://www.hey.com/features/),
  [Superhuman](https://help.superhuman.com/hc/en-us/articles/46005666142733-Remind-Me))
- **The conversation opens where a reply is needed.** At rest an email's
  page is that email (parts.go). Your turn to reply opens it with
  `?show=conversation`: every email oldest first, each under a heading of
  who and when, so a screen reader moves through them by heading (WCAG
  1.3.1, 2.4.6). Each says only its own words; the email the page is about
  is said once, at the top. An agent reads the same with look, or lists
  `thread=<id>`.

## Not done, and why

- **OAuth sign-in.** It needs Sameway registered with each provider and a
  web address to come back to; a program on the person's own computer has
  neither. App passwords work for Gmail, iCloud, Yahoo and Fastmail.
- **Remind me if no reply.** Waiting on them is a count on Today with how
  long each has waited, which is enough to notice without another thing
  to set.
- **Subject matching for mail without headers.** Two emails both called
  "Re: Quote" are often different conversations; a missing reference is
  left as its own email rather than guessed into a thread.
- **Fetching pictures through a proxy, as HEY does.** It would mean
  Sameway fetching from the sender's server on the person's behalf, which
  still tells the sender when; the words are what the person forwarded
  the email for.
