# Sorting: tags and suggestions an action gives

An action can tag what sets it off (classify) and suggest a record from it
(suggest). Both act for the person while they are away. These are the rules
for how that shows on Today, so it helps without surprising anyone.

## Why it works this way

- **A suggestion until the person decides.** A tag an action gives waits as
  a question: Keep, Take it off, or Change to. A suggested record waits as
  Make it, Change or No. Nothing is moved or hidden by a tag; it is a word on
  the record. Superhuman's auto labels work the same way, tagging without
  moving mail, so the person's own way of working stays as it was.
- **Each record in one place.** A tag on an email waiting to be sorted is
  checked in that email's row under To sort, above what was suggested from
  it. It is not asked again under Tags to check, which holds the tags on
  records not waiting to be sorted. A screen reader hears the email once,
  then everything about it.
- **An answer to one question answers the one behind it.** Making a task
  suggested from an email tagged *to do* says the email was to do, so that
  tag is kept as the person's own choice and not asked about again. Saying
  No to the task says nothing about the tag (it may be to do, and not a
  task), so the tag is left to check.
- **Acting for the person is never silent.** A tag the action kept for them,
  because their past choices plainly settle it, is not asked about, but it is
  shown, folded under *Kept for you from your choices* with how many, for a
  week. Each says why, and has Take it off and Change to (not Keep, which it
  already is). What needs nothing is folded the same way under its tag's
  name. Folded, these do not compete for attention; shown, nothing happens
  behind the person's back.
- **Every answer teaches, and says so.** Keep, Take it off and Change each
  end with "the action follows that next time", as Turned down does for a
  suggestion. Learning is from what the person does anyway, as SaneBox
  learns from mail moved between folders and Gmail from a message moved to
  another tab, not from a separate training step.
- **Why is said beside the tag, in a few words.** The why is the model's own
  words for one record, said as such ("Tagged to do: asks for a payment"),
  not offered as proof. A kept tag's why names the past choice it followed.
- **Two kinds of sure, not a number.** *Suggested* (Tagged) and *kept for
  you* (Kept) are the only confidence shown: each says what the person
  should do, which is what a confidence level is for.
- **Agents read and do the same.** Today is read through look_at_page and
  GET /api/look with the same words; taking a tag off or changing it is
  update_record on the record's tags, which is noticed the same way. Keeping
  a tag and turning a suggestion down are the person's alone (routes), being
  the judgement the action then follows.

## Sources

- Microsoft, Guidelines for Human-AI Interaction (Amershi et al., CHI 2019):
  support efficient correction, make clear why the system did what it did,
  convey the consequences of user actions.
  https://www.microsoft.com/en-us/research/publication/guidelines-for-human-ai-interaction/
- Google PAIR, Explainability + Trust: show confidence only where it
  changes what a person does, and categories rather than numbers that need
  explaining; examples as explanation.
  https://pair.withgoogle.com/guidebook-v2/chapter/explainability-trust/
- NN/g, Explainable AI in chat interfaces: a model's account of its reasons
  may not be how it decided; do not present it as certain.
  https://www.nngroup.com/articles/explainable-ai/
- SaneBox, training by moving mail, and fixing a mistake by moving it back.
  https://www.sanebox.com/help/140-how-do-i-train-teach-sanebox,
  https://www.sanebox.com/help/163-what-if-i-make-a-mistake-training
- Gmail categories: a message moved to another tab, with a prompt to do the
  same for that sender. https://kb.swarthmore.edu/wiki/Gmail_Categories_and_Inbox_Tabs
- Superhuman auto labels: tagged, not moved.
  https://techcrunch.com/2025/02/19/superhuman-introduces-ai-powered-categorization-to-reduce-spammy-emails-in-your-inbox/
- HEY's Screener: one decision a sender, made once.
  https://www.hey.com/features/the-screener/
- W3C COGA, Making Content Usable: provide feedback, make it easy to undo,
  no unexpected changes. https://www.w3.org/TR/coga-usable/
- WCAG 2.2 Understanding 3.2.5 Change on Request.
  https://www.w3.org/WAI/WCAG22/Understanding/change-on-request.html

## Not done, and why

- **A confidence number.** The small local model has no calibrated
  confidence, and a number would need explaining. Suggested and kept say
  what to do.
- **Gmail's "do this for future messages?"** Every answer is already
  learned from and says so; a second question after each answer is a chore.
- **HEY's sender rules.** A tag is decided by what a record says and what the
  tag means, not who sent it; a newsletter and a bill can come from one bank.
- **Moving or hiding what a tag is on.** Tags only add a word. What needs
  nothing is folded on Today, never taken away.
- **Naming the exact past record a tag followed.** The model is shown the
  latest choices together, newest first, and does not reliably say which it
  followed; the kept tag's why is its own words.
- **No to a suggestion taking the tag off.** Not wanting a task does not
  mean nothing was to do.
