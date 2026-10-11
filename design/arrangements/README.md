# Arrangements and Templates

An arrangement is a page a job wants: which blocks, in what order, how
wide. The assistant lays one out with `add_arrangement`. A person gets the
same pages from Templates (`/templates`), where one press makes the kinds
of thing the page keeps, a tab named for it, and its blocks there
(`chat.UseTemplate`).

## Why it works this way

- **Your own things, never example words.** Every block shows the
  person's records and says so when there are none (#380). Sample rows
  are the commonest complaint about other products' templates: Coda users
  ask how to get rid of the sample people a template brought
  ([Coda community](https://community.coda.io/t/remove-sample-users/15777)),
  and Airtable has a "Clear data" step for the same reason
  ([Airtable](https://support.airtable.com/docs/using-airtable-templates)).
- **Said in words before it is made.** Notion and Airtable show a preview
  before you copy
  ([Notion](https://www.notion.com/help/guides/the-ultimate-guide-to-notion-templates)).
  A picture of an empty page says little and nothing to a screen reader,
  so each template says what its tab shows, by the blocks' own headings
  ("Its tab shows: Still to do, Dates and Done."), and which kinds of
  thing it adds. Specific words let people judge fit quickly
  ([NN/g, use-case suggestions](https://www.nngroup.com/articles/designing-use-case-prompt-suggestions/)).
- **Filed as an index files them.** Eleven templates is a list to scan,
  so they are in the order of their titles, with a leading "The" left out
  (The move under M). Each is a heading, so a screen reader can move from
  one to the next.
- **Said back, and one Undo.** After a press the page you land on says
  what was made ("Job search is a tab of its own now. It keeps
  applications.") with Undo beside it
  ([GOV.UK confirmation pages](https://design-system.service.gov.uk/patterns/confirmation-pages/),
  [COGA 4.5.10 provide feedback](https://www.w3.org/TR/coga-usable/)).
  The tab and its blocks are one entry in Activity, so one Undo takes the
  whole page back ([COGA 4.5.2 let users go back](https://www.w3.org/TR/coga-usable/)).
- **Used twice, said and named.** A template whose tab is already there
  links to it and says another use makes another. The second is named
  "Job search 2", the first free number.
- **The same for an agent.** An agent presses the same button through
  look (`POST /api/look` with the path `/templates/use`) and reads, where
  it lands, the words a person is told. Its own tools make the same page
  step by step: `add_type`, `create_canvas`, `add_arrangement`.

## Not done, and why

- **No sample data, even as an option.** Trello lets you keep a
  template's cards; here a block can only show records, and invented ones
  would sit among the person's own (#380).
- **No screenshots or live previews.** They would be empty pages, and
  costly to keep true; the words say the same to everyone.
- **No categories or search.** Eleven templates fit one screen of
  headings; grouping them would add a choice before the choice
  ([COGA 4.6.3 avoid too much content](https://www.w3.org/TR/coga-usable/)).
- **Undo leaves the kinds it made.** A kind may already hold records
  added since, and an empty one costs nothing; it can be removed by hand.
